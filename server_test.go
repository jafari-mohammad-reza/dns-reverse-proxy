package main

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func waitForPort(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("udp", address, 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return ErrTimeout
}

var ErrTimeout = &net.OpError{Op: "dial", Net: "udp", Addr: nil, Err: timeoutErr{}}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout waiting for port" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestServerHandleRequest(t *testing.T) {
	conf, err := InitConf()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	server := NewServer(conf)

	go func() {
		if err := server.Start(); err != nil {
			t.Errorf("server exited with error: %v", err)
		}
	}()

	addr := "127.0.0.1:5354"
	if err := waitForPort(addr, 5*time.Second); err != nil {
		t.Fatalf("DNS server not reachable at %s: %v", addr, err)
	}

	t.Run("query A record for google.com", func(t *testing.T) {
		c := &dns.Client{
			Net:     "udp",
			Timeout: 2 * time.Second,
		}

		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("google.com"), dns.TypeA)

		r, rtt, err := c.Exchange(m, addr)
		if err != nil {
			t.Fatalf("DNS query failed: %v", err)
		}

		t.Logf("Query RTT: %s", rtt)

		if r.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS query returned non-success Rcode: %d", r.Rcode)
		}

		found := false
		for _, a := range r.Answer {
			if arec, ok := a.(*dns.A); ok {
				t.Logf("Response: google.com → %s", arec.A)
				found = true
			}
		}
		if !found {
			t.Errorf("No A records found in DNS response")
		}
	})
}
