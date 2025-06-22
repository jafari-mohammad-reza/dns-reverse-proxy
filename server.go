package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/miekg/dns"
)

type Server struct {
	conf            *Conf
	dnsServer       *dns.Server
	domainResolvers map[string][]string
	logger          ILogger
}

func NewServer(conf *Conf, logger ILogger) *Server {
	domainResolvers := make(map[string][]string, len(conf.DomainResolvers))
	for _, resolver := range conf.DomainResolvers {
		domainResolvers[fmt.Sprintf("%s.", resolver.Domain)] = resolver.Resolvers
	}
	return &Server{
		conf:            conf,
		domainResolvers: domainResolvers,
		logger:          logger,
	}
}

func (s *Server) Start() error {
	listenAddr := fmt.Sprintf(":%d", s.conf.Port)
	s.dnsServer = &dns.Server{
		Addr:    listenAddr,
		Net:     "udp",
		Handler: dns.HandlerFunc(s.handleDNSRequest),
	}
	log.Printf("[+] Starting DNS proxy on %s", listenAddr)
	if err := s.dnsServer.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
func (s *Server) Stop() error {
	return s.dnsServer.Shutdown()
}

func (s *Server) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	start := time.Now()
	fmt.Println(r.Question)
	if len(r.Question) == 0 {
		log.Println("[-] Empty DNS question received, ignoring")
		return
	}
	clientIp, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	domain := r.Question[0]
	log.Printf("[>] Query: %s (%s)", domain.Name, dns.TypeToString[domain.Qtype])
	resp, err := s.resolveWithFallback(r, clientIp)
	if err != nil {
		log.Printf("[-] All upstreams failed for %s: %v", domain.Name, err)
		m := new(dns.Msg)
		m.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(m)
		return
	}
	for _, ans := range resp.Answer {
		if a, ok := ans.(*dns.A); ok {
			log.Printf("[<] Response: %s → %s", domain.Name, a.A.String())
		}
	}

	err = w.WriteMsg(resp)
	if err != nil {
		log.Printf("[-] Failed to write response: %v", err)
	}

	log.Printf("[✓] Resolved %s in %s", domain.Name, time.Since(start))
}
func (s *Server) resolveWithFallback(r *dns.Msg, clientIp string) (*dns.Msg, error) {
	c := &dns.Client{
		Net:     "udp",
		Timeout: 2 * time.Second,
	}
	domainName := r.Question[0].Name
	if resolvers, ok := s.domainResolvers[domainName]; ok {
		for _, upstream := range resolvers {
			return s.resolverDomain(r, c, domainName, upstream, clientIp)
		}
	}
	for _, upstream := range s.conf.UpstreamAddrs {
		return s.resolverDomain(r, c, domainName, upstream, clientIp)
	}

	return nil, dns.ErrConnEmpty
}
func (s *Server) resolverDomain(r *dns.Msg, c *dns.Client, domainName, upstream, clientIp string) (*dns.Msg, error) {
	log.Printf("[>] Querying upstream: %s for %s", upstream, domainName)
	resp, _, err := c.Exchange(r, upstream)
	if err != nil {
		log.Printf("[-] Upstream %s failed: %v", upstream, err)
		return nil, errors.New("upstream query failed")
	}
	s.logger.Info(Log{
		Time:     time.Now().Format(time.RFC3339Nano),
		Level:    Info,
		Domain:   domainName,
		ClientIp: clientIp,
		Qtype:    dns.TypeToString[r.Question[0].Qtype],
		Resolver: upstream,
	})
	if resp != nil && resp.Rcode == dns.RcodeSuccess {
		return resp, nil
	}
	return nil, fmt.Errorf("[-] Upstream %s returned non-success Rcode: %d", upstream, resp.Rcode)

}
