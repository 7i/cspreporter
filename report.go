package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"time"
)

type cspReport struct {
	R report `json:"csp-report"`
}

type report struct {
	BlockedURI         string `json:"blocked-uri"`
	DocumentURI        string `json:"document-uri"`
	LineNumber         int    `json:"line-number"`
	OriginalPolicy     string `json:"original-policy"`
	Referrer           string `json:"referrer"`
	ScriptSample       string `json:"script-sample"`
	SourceFile         string `json:"source-file"`
	ViolatedDirective  string `json:"violated-directive"`
	EffectiveDirective string `json:"effective-directive"`
	Disposition        string `json:"disposition"`
	StatusCode         int    `json:"status-code"`
	ColumnNumber       int    `json:"column-number"`
}

func reportSrv(w http.ResponseWriter, req *http.Request) {
	req.Body = http.MaxBytesReader(w, req.Body, globalConfig.MaxCSPReportSize)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return // Skip if errors
	}
	var report cspReport
	err = json.Unmarshal(body, &report)
	if err != nil {
		return // Skip if errors
	}
	u, err := url.Parse(report.R.DocumentURI)
	if err != nil {
		return // Skip if errors
	}

	d, ok := globalDomainMap[filepath.Base(u.Hostname())]
	if ok {
		if !globalConfig.Silent {
			fmt.Println(string(body), "\n", filepath.Base(u.Hostname()), "\n\n")
		}
		d.mutex.Lock()
		if globalConfig.Syslog != "" {
			go sendSyslogMessage(d.name, string(body))
		}
		d.textInZip.Write(body)
		d.textInZip.Write([]byte("\n"))
		d.nr++
		if d.nr > globalConfig.MaxReportsPerZip {
			d.mutex.Unlock()
			d.flush()
		} else {
			d.mutex.Unlock()
		}
	}
}

func sendSyslogMessage(name, body string) {
	sysLog, err := Dial(globalConfig.Transport, globalConfig.Syslog, LOG_WARNING|LOG_DAEMON, name)
	if err != nil {
		if !globalConfig.Silent {
			log.Println(err)
		}
	} else {
		fmt.Fprintf(sysLog, "CSP report from Domain "+name+" : "+body)
	}
}

func cspReportListener() {
	s := &http.Server{
		Addr:           globalConfig.ReportURI,
		Handler:        http.HandlerFunc(reportSrv),
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   5 * time.Second,
		MaxHeaderBytes: int(globalConfig.MaxCSPReportSize),
	}
	log.Fatalln(s.ListenAndServe())
}

///////////////////////////////////////////////////////////////////////////////
///Test Code///////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////
//func testSyslogHandler(c net.Conn) {
//	defer c.Close()
//	buf := make([]byte, 4096)
//	fmt.Println("SYSLOG:")
//	for {
//		n, err := c.Read(buf)
//		if err != nil {
//			if err != io.EOF {
//				fmt.Println("read error:", err)
//			}
//			break
//		}
//		fmt.Println(string(buf[:n]))
//	}
//}
//
//func testSyslog() {
//	fmt.Println("Starting testSyslog")
//	ln, err := net.Listen("tcp", ":8282")
//	if err != nil {
//		log.Fatalln("cant start testSyslog")
//	}
//	for {
//		conn, err := ln.Accept()
//		if err != nil {
//			continue
//		}
//		go testSyslogHandler(conn)
//	}
//}
