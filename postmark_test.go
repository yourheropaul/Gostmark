package postmark

import (
    "fmt"
    "testing"
)

func TestPMMail(t *testing.T) {
    p := CreatePMMail("1234567")
    p.Sender = "Dave Martorana <themartorana@yahoo.com>"
    p.To = "Dave Martorrrrana <dave@flyclops.com>"
    p.Subject = "This is a test"
    p.TextBody = "This is a test"
    p.HTMLBody = "<strong>This is a test</strong>"

    p.AddCustomHeader("X-H1", "Dave Rulez")
    if err := p.AddAttachment("postmark.go"); err != nil {
        t.Errorf("Error attaching file: %s\n", err)
        t.Fail()
    }

    if packet, err := p.MessageAsJSONPacket(); err != nil {
        t.Errorf("Trouble getting JSON packet: %s\n", err)
        t.Fail()
    } else {
        fmt.Println(string(packet))
    }
}
