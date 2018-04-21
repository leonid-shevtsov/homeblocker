package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
	yaml "gopkg.in/yaml.v2"
)

type interval struct {
	wildcard bool
	from     int
	to       int
}

type scheduleLine struct {
	enabled bool
	minute  interval
	hour    interval
	day     interval
	month   interval
	weekday interval
}

type schedule []scheduleLine

// Block is a group of sites to block
type Block struct {
	Domains         []string `yaml:"domains"`
	WildcardDomains []string `yaml:"wildcard_domains"`
	Schedule        string   `yaml:"schedule"`
}

// Configuration is the type loaded from homeblocker.yml
type Configuration struct {
	Port     int              `yaml:"port"`
	Upstream string           `yaml:"upstream"`
	Blocks   map[string]Block `yaml:"blocks"`
}

func parseInterval(text string) interval {
	if text == "*" {
		return interval{wildcard: true}
	}
	if strings.Contains(text, "-") {
		fromTo := strings.Split(text, "-")
		if len(fromTo) != 2 {
			log.Panicf("Interval must be from-to: %s", text)
		}
		from, err := strconv.Atoi(fromTo[0])
		if err != nil {
			log.Panicf("Interval must be from-to: %s", text)
		}
		to, err := strconv.Atoi(fromTo[1])
		if err != nil {
			log.Panicf("Interval must be from-to: %s", text)
		}
		return interval{from: from, to: to}
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		log.Panicf("Field must be either a wildcard *, a number, or an interval: %s", text)
	}
	return interval{from: value, to: value + 1}
}

func parseScheduleLine(lineText string) scheduleLine {
	parts := strings.Split(lineText, " ")
	if len(parts) != 6 {
		log.Panicf("Schedule lines must follow the crontab format; 6 fields separated with spaces: %s", lineText)
	}
	var line scheduleLine
	line.minute = parseInterval(parts[0])
	line.hour = parseInterval(parts[1])
	line.day = parseInterval(parts[2])
	line.month = parseInterval(parts[3])
	line.weekday = parseInterval(parts[4])
	switch parts[5] {
	case "on":
		line.enabled = true
	case "off":
		line.enabled = false
	default:
		log.Panicf("Schedule line must end with either \"on\" or \"off\": %s", lineText)
	}
	return line
}

func parseSchedule(scheduleText string) schedule {
	var schedule schedule
	lines := strings.Split(scheduleText, "\n")
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) > 0 {
			schedule = append(schedule, parseScheduleLine(line))
		}
	}
	return schedule
}

func loadConfig() Configuration {
	file, err := os.Open("homeblocker.yml")
	if err != nil {
		panic(err)
	}
	decoder := yaml.NewDecoder(file)
	var configuration Configuration
	err = decoder.Decode(&configuration)
	if err != nil {
		panic(err)
	}
	return configuration
}

type handlerBlock struct {
	schedule        schedule
	domains         map[string]bool
	wildcardDomains []string
}

type handler struct {
	upstream string
	blocks   []*handlerBlock
}

func (interval interval) matches(number int) bool {
	return interval.wildcard || (number >= interval.from && number < interval.to)
}

func (line scheduleLine) matches(time time.Time) bool {
	return line.minute.matches(time.Minute()) &&
		line.hour.matches(time.Hour()) &&
		line.day.matches(time.Day()) &&
		line.month.matches(int(time.Month())) &&
		line.weekday.matches(int(time.Weekday()))
}

func (schedule schedule) isEnabled(current time.Time) bool {
	isEnabled := true // default
	for _, line := range schedule {
		if line.matches(current) {
			isEnabled = line.enabled
		}
	}
	return isEnabled
}

func parseBlock(configBlock Block) *handlerBlock {
	block := new(handlerBlock)
	block.schedule = parseSchedule(configBlock.Schedule)
	block.domains = make(map[string]bool)
	for _, domain := range configBlock.Domains {
		block.domains[domain+"."] = true
		block.domains["www."+domain+"."] = true
	}
	for _, domain := range configBlock.WildcardDomains {
		block.wildcardDomains = append(block.wildcardDomains, "."+domain+".")
	}
	return block
}

func newHandler(config Configuration) *handler {
	handler := &handler{upstream: config.Upstream}
	if !strings.Contains(handler.upstream, ":") {
		handler.upstream = handler.upstream + ":53"
	}
	for _, block := range config.Blocks {
		handler.blocks = append(handler.blocks, parseBlock(block))
	}
	return handler
}

func (block *handlerBlock) matchesWildcardDomain(domain string) bool {
	for _, wildcard := range block.wildcardDomains {
		if strings.HasSuffix(domain, wildcard) {
			return true
		}
	}
	return false
}

func (h *handler) isBlocked(domain string, time time.Time) bool {
	for _, block := range h.blocks {
		if block.schedule.isEnabled(time) && (block.domains[domain] || block.matchesWildcardDomain(domain)) {
			return true
		}
	}
	return false
}

func (h *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {

	if r.Question[0].Qtype == dns.TypeA {
		domain := r.Question[0].Name
		addrParts := strings.Split(w.RemoteAddr().String(), ":")
		log.Printf("Resolving type A request from %v for domain %s", addrParts[0], domain)
		if h.isBlocked(domain, time.Now()) {

			msg := dns.Msg{}
			msg.SetReply(r)
			msg.Authoritative = true
			w.WriteMsg(&msg)
			// blocked -> empty response
			log.Printf("...Blocked")
			return
		}
	}
	// fallback to upstream server
	c := new(dns.Client)
	in, _, err := c.Exchange(r, h.upstream)
	if err != nil {
		msg := dns.Msg{}
		msg.SetReply(r)
		log.Printf("Error resolving domain: %v", err)
		w.WriteMsg(&msg)
		return
	}

	w.WriteMsg(in)
}

func main() {
	config := loadConfig()
	if config.Port == 0 {
		config.Port = 53
	}

	bindAddress := fmt.Sprintf(":%d", config.Port)
	srv := &dns.Server{Addr: bindAddress, Net: "udp"}

	srv.Handler = newHandler(config)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
