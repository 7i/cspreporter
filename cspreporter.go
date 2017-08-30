package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"text/template"
)

type configuration struct {
	ZipPageURI       string
	ReportURI        string
	DomainsWhitelist []string
	Syslog           string
	Transport        string
	MaxReportsPerZip int64
	ZipsDir          string
	TemplateDir      string
	ZipPageCSPDir    string
	MaxCSPReportSize int64
	Silent           bool
}

var globalConfig configuration
var globalMainpageTemplate *template.Template
var globalCSPTemplate *template.Template
var globalDomainMap map[string]*domain

func main() {
	// setup sets global variables, applys configs and sets defaut values
	setup()

	// Populate the globalDomainMap and register handlers for each domain
	for _, domainName := range globalConfig.DomainsWhitelist {
		domain := newDomain(domainName)
		globalDomainMap[domainName] = domain

		http.Handle("/flush/"+domainName+"/", domain)

		domainServer, err := newDomainPageServer(domainName)
		if err != nil {
			log.Fatalln(err)
		}
		http.Handle("/domain/"+domainName+"/", domainServer)
	}

	// Start cspReportListener in a new go rutinel
	go cspReportListener()

	http.Handle("/get/", http.HandlerFunc(getZipServer))
	http.Handle("/del/", http.HandlerFunc(delZipServer))
	http.Handle("/", http.HandlerFunc(mainpageServer))

	log.Fatalln(http.ListenAndServe(globalConfig.ZipPageURI, nil))
}

// setup reads config from file conf, sets default values and verifies that
// critical config parameters exsist
func setup() {
	conf := flag.String("conf", "cspreporter.conf", "Path to the CSP Reporter config file.")
	flag.Parse()

	globalDomainMap = make(map[string]*domain)
	jsonData, err := ioutil.ReadFile(*conf)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(jsonData, &globalConfig)
	if err != nil {
		log.Fatalln(err)
	}

	// Verify that critical config parameters exsist
	if len(globalConfig.DomainsWhitelist) == 0 {
		log.Fatalln("Invalid config, needs at least one whitelisted domain in parameter DomainsWhitelist")
	}
	if globalConfig.ReportURI == "" {
		log.Fatalln("Invalid config, missing parameter ReportUri (DNS adress and port to this servers ReportHandler)")
	}
	if globalConfig.ZipPageURI == "" {
		log.Fatalln("Invalid config, missing parameter ZipPageURI (DNS adress and port to this servers Zip download page)")
	}

	// Specify default values for non critical config parameters
	if globalConfig.MaxCSPReportSize == 0 {
		globalConfig.MaxCSPReportSize = 1 << 16
	}
	if globalConfig.Transport == "" {
		globalConfig.Transport = "tcp"
	}
	if globalConfig.Syslog == "" && !globalConfig.Silent {
		log.Println("Missing parameter Syslog, inactivating syslog messanges")
	}
	if globalConfig.ZipsDir == "" {
		if !globalConfig.Silent {
			log.Println("Missing parameter ZipsDir (where to save and read zip files), defaulting to current directory")
		}
		globalConfig.ZipsDir = "./"
	}
	if globalConfig.TemplateDir == "" && !globalConfig.Silent {
		log.Println("Missing parameter TemplateDir, defaulting to current directory")
		globalConfig.TemplateDir = "./"
	}

	// Ensure all paths end with "/"
	if !strings.HasSuffix(globalConfig.TemplateDir, "/") {
		globalConfig.TemplateDir += "/"
	}
	if !strings.HasSuffix(globalConfig.ZipsDir, "/") {
		globalConfig.ZipsDir += "/"
	}

	// Populate globalCSPTemplate eather from defaultCSPTemplate or from the
	// csp.tmpl file
	defaultCSPTemplate := "default-src 'none'; script-src 'nonce-{{.}}; style-src 'none'; media-src 'none'; img-src 'self' data:; child-src 'none'; frame-src 'none'; frame-ancestors 'none'; object-src 'none'; base-uri 'none'; font-src 'none'; connect-src 'none'; report-uri https://{{.}}/csp;"
	csp, err := ioutil.ReadFile(globalConfig.TemplateDir + "csp.tmpl")
	if err != nil {
		globalCSPTemplate = template.Must(template.New("csp").Parse(defaultCSPTemplate))
	}
	cspTemplate := string(bytes.Replace(csp, []byte("\n"), []byte(" "), -1))
	globalCSPTemplate = template.Must(template.New("csp").Parse(cspTemplate))

	globalMainpageTemplate = template.Must(template.ParseFiles(globalConfig.TemplateDir + "index.tmpl"))
}
