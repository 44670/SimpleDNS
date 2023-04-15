package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type Config struct {
	DoHURL string            `json:"dohurl"`
	Rules  map[string]string `json:"rules"`
}

type IPCacheEntry struct {
	IP       string
	ExpireAt time.Time
}

var ipCache sync.Map
var matchDomainMap = make(map[string]string)
var matchDomainAndSubDomainMap = make(map[string]string)

var dohHttpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	splitRules(config.Rules)

	server := &dns.Server{
		Addr:    ":53",
		Net:     "udp",
		Handler: &dnsHandler{dohURL: config.DoHURL},
	}

	log.Printf("Starting DNS server on %s\n", server.Addr)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func splitRules(rules map[string]string) {
	for domain, ip := range rules {
		if strings.HasPrefix(domain, "*.") {
			matchDomainAndSubDomainMap[strings.TrimPrefix(domain, "*.")] = ip
		} else {
			matchDomainMap[domain] = ip
		}
	}
}

type dnsHandler struct {
	dohURL string
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	msg.Authoritative = true

	for _, question := range r.Question {
		switch question.Qtype {
		case dns.TypeA:
			ip, from := resolveDomain(h.dohURL, question.Name)
			log.Printf("Resolved domain: %s, IP: %s (from %s)\n", question.Name, ip, from)
			if ip != "" {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: net.ParseIP(ip),
				}
				msg.Answer = append(msg.Answer, rr)
			}
		}
	}

	w.WriteMsg(&msg)
}

func resolveDomain(dohURL, domain string) (string, string) {
	domain = strings.TrimSuffix(domain, ".")
	if ip, ok := matchDomainMap[domain]; ok {
		return ip, "domain rule"
	}

	subDomain := domain
	for {
		if ip, ok := matchDomainAndSubDomainMap[subDomain]; ok {
			return ip, "subdomain rule"
		}
		split := strings.SplitN(subDomain, ".", 2)
		if len(split) < 2 {
			break
		}
		subDomain = split[1]
	}
	cacheKey := domain
	if entry, ok := ipCache.Load(cacheKey); ok {
		ipEntry := entry.(IPCacheEntry)
		if time.Now().Before(ipEntry.ExpireAt) {
			return ipEntry.IP, "cache"
		}
	}

	ret := resolveDomainOverDoH(dohURL, domain)
	if ret != "" {
		ipCache.Store(cacheKey, IPCacheEntry{
			IP:       ret,
			ExpireAt: time.Now().Add(10 * time.Minute),
		})
		return ret, "DoH"
	} else {
		log.Printf("Error resolving domain: %s\n", domain)
	}
	return "", ""
}

func resolveDomainOverDoH(dohURL, domain string) string {
	req, err := http.NewRequest("GET", dohURL, nil)
	if err != nil {
		log.Printf("Error creating DoH request: %v\n", err)
		return ""
	}

	q := req.URL.Query()
	q.Add("name", domain)
	q.Add("type", "A")
	req.URL.RawQuery = q.Encode()

	resp, err := dohHttpClient.Do(req)
	if err != nil {
		log.Printf("Error performing DoH request: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error response from DoH server: %s\n", resp.Status)
		return ""
	}

	var dohResponse struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}

	err = json.NewDecoder(resp.Body).Decode(&dohResponse)
	if err != nil {
		log.Printf("Error decoding DoH response: %v\n", err)
		return ""
	}

	for _, answer := range dohResponse.Answer {
		if answer.Type == int(dns.TypeA) {
			return answer.Data
		}
	}

	return ""
}
