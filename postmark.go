// Go package for postmarkapp.com
package postmark

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path"
)

const __POSTMARK_URL__ string = "https://api.postmarkapp.com/email"
const __VERSION__ string = "0.1"

type PMMail struct {
	userAgent string
	apiKey    string

	customHeaders []header
	attachments   []attachment

	Sender   string
	ReplyTo  string
	To       string
	CC       string
	BCC      string
	Subject  string
	Tag      string
	HTMLBody string
	TextBody string
}

type header struct {
	Name  string
	Value string
}

type attachment struct {
	Name        string
	Content     string
	ContentType string
}

type Reply struct {
	ErrorCode   int
	Message     string
	MessageID   string
	SubmittedAt string
	To          string
}

// Create a new PMMail struct with
// an Postmark API key, and return a
// pointer to it
func CreatePMMail(apikey string) *PMMail {
	pmmail := &PMMail{apiKey: apikey}
	pmmail.userAgent = fmt.Sprintf("Go (Go postmark package library version %f)", __VERSION__)

	return pmmail
}

// Add a custom header to the email message
func (p *PMMail) AddCustomHeader(name, value string) {
	h := header{
		Name:  name,
		Value: value,
	}
	p.customHeaders = append(p.customHeaders, h)
}

// Add a file attachment by file path
// Most shamefully inspired by
// https://github.com/gcmurphy/postmark/blob/master/message.go
func (p *PMMail) AddAttachment(file string) error {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}
	if fileInfo.Size() > int64(10e6) {
		return fmt.Errorf("File size %d exceeds 10MB limit.", fileInfo.Size())
	}

	fileHandle, err := os.Open(file)
	if err != nil {
		return err
	}

	content, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		fileHandle.Close()
		return err
	} else {
		fileHandle.Close()
	}

	mimeType := mime.TypeByExtension(path.Ext(file))
	if len(mimeType) == 0 {
		mimeType = "application/octet-stream"
	}

	a := attachment{
		Name:        fileInfo.Name(),
		Content:     base64.StdEncoding.EncodeToString(content),
		ContentType: mimeType,
	}
	p.attachments = append(p.attachments, a)

	return nil
}

func (p *PMMail) checkValues() error {
	if p.Sender == "" {
		return fmt.Errorf("Cannot send e-mail without a sender (.Sender field)")
	}
	if p.To == "" {
		return fmt.Errorf("Cannot send e-mail without recipient (.To field)")
	}
	if p.Subject == "" {
		return fmt.Errorf("Cannot send e-mail without a subject (.Subject field)")
	}
	if p.HTMLBody == "" && p.TextBody == "" {
		return fmt.Errorf("Cannot send email without an HTML body, text body or both")
	}

	return nil
}

func (p *PMMail) createJsonMessagePacket() ([]byte, error) {
	if err := p.checkValues(); err != nil {
		return []byte{}, err
	}

	json_interface := map[string]interface{}{
		"From":    p.Sender,
		"To":      p.To,
		"Subject": p.Subject,
	}

	if p.ReplyTo != "" {
		json_interface["ReplyTo"] = p.ReplyTo
	}

	if p.CC != "" {
		json_interface["Cc"] = p.CC
	}

	if p.BCC != "" {
		json_interface["Bcc"] = p.BCC
	}

	if p.Tag != "" {
		json_interface["Tag"] = p.Tag
	}

	if p.HTMLBody != "" {
		json_interface["HtmlBody"] = p.HTMLBody
	}

	if p.TextBody != "" {
		json_interface["TextBody"] = p.TextBody
	}

	if i := len(p.attachments); i > 0 {
		json_interface["Attachments"] = p.attachments
	}

	if i := len(p.customHeaders); i > 0 {
		json_interface["Headers"] = p.customHeaders
	}

	return json.Marshal(json_interface)
}

// Returns the compiled Postmark API
// formatted JSON packet to send to
// Postmark
func (p *PMMail) MessageAsJSONPacket() ([]byte, error) {
	if json_message, err := p.createJsonMessagePacket(); err != nil {
		return []byte{}, err
	} else {
		return json_message, nil
	}
}

// Attempts to send the email by connecting to
// Postmark's servers and sending the
// formatted JSON packet
func (p *PMMail) Send() (*Reply, error) {

	data, err := p.MessageAsJSONPacket()

	if err != nil {
		return nil, err
	}

	badata := bytes.NewBuffer(data)
	request, err := http.NewRequest("POST", __POSTMARK_URL__, badata)

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Postmark-Server-Token", p.apiKey)

	// Set any custom headers
	for _, h := range p.customHeaders {
		request.Header.Set(h.Name, h.Value)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}

	switch {
	case response.StatusCode == 401:
		return nil, fmt.Errorf("[Postmark] HTTP error %d : Missing headers", response.StatusCode)
	case response.StatusCode == 404:
		return nil, fmt.Errorf("[Postmark] HTTP error %d : Page not found", response.StatusCode)
	case response.StatusCode == 422:
		return nil, fmt.Errorf("[Postmark] HTTP error %d : Bad JSON", response.StatusCode)
	case response.StatusCode == 500:
		return nil, fmt.Errorf("[Postmark] HTTP error %d : Server error", response.StatusCode)
	}

	var body bytes.Buffer
	_, err = io.Copy(&body, response.Body)
	response.Body.Close()
	if err != nil {
		return nil, err
	}

	reply := new(Reply)
	json.Unmarshal([]byte(body.String()), reply)

	if err != nil {
		return nil, err
	}

	if reply.ErrorCode != 0 {
		return reply, fmt.Errorf("Error Code: %d", reply.ErrorCode)
	}

	// Send
	return reply, nil
}
