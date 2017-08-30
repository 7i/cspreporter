package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// mainpageServer serves the all requests to /
func mainpageServer(w http.ResponseWriter, req *http.Request) {
	_, csp, err := getCSP()
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	w.Header().Add("Content-Security-Policy", csp)

	err = globalMainpageTemplate.Execute(w, globalConfig.DomainsWhitelist)

	if err != nil {
		log.Println(err)
		http.NotFound(w, req)
		return
	}
}

type domainPageServer struct {
	Page *template.Template
	Name string
}

// newDomainPageServer returns a domainPageServer that uses domain.tmpl and name as domainPageServer.Name
func newDomainPageServer(name string) (srv *domainPageServer, err error) {
	srv = new(domainPageServer)
	srv.Page, err = template.ParseFiles(globalConfig.TemplateDir + "domain.tmpl")
	if err != nil {
		return nil, err
	}
	srv.Name = name
	return srv, nil
}

type domainPage struct {
	Name    string
	Nr      int64
	ZipList []zipInfo
	Nonce   string
}

func (srv *domainPageServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	zipList, err := getZipList(srv.Name)
	if err != nil {
		http.Error(w, "", 500)
		return
	}
	nonce, csp, err := getCSP()
	if err != nil {
		http.Error(w, "", 500)
		return
	}
	var dp domainPage
	domain, ok := globalDomainMap[srv.Name]
	if ok {
		domain.mutex.Lock()
		nr := domain.nr
		domain.mutex.Unlock()
		dp = domainPage{Name: srv.Name, Nr: nr, ZipList: zipList, Nonce: nonce}
	} else {
		dp = domainPage{Name: srv.Name, ZipList: zipList, Nonce: nonce}
	}

	w.Header().Add("Content-Security-Policy", csp)

	srv.Page.Execute(w, dp)
}

// getZipServer handles requests to /get/ and serves .zip files from the globalConfig.ZipPageURI directory
func getZipServer(w http.ResponseWriter, req *http.Request) {

	file := fileNameFromURL(req.URL.Path)

	ref, err := url.Parse(req.Referer())
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	if filepath.Ext(file) == ".zip" && ref.Host == globalConfig.ZipPageURI {
		http.ServeFile(w, req, globalConfig.ZipsDir+file)
	} else {
		http.Error(w, "", 500)
	}
}

// DelZip handles requests to /del/ and deletes .zip files from the globalConfig.ZipPageURI directory
func delZipServer(w http.ResponseWriter, req *http.Request) {
	file := fileNameFromURL(req.URL.Path)

	ref, err := url.Parse(req.Referer())
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	if filepath.Ext(file) == ".zip" && ref.Host == globalConfig.ZipPageURI {
		os.Remove(globalConfig.ZipsDir + file)
		http.Redirect(w, req, req.Referer(), 303)
	} else {
		http.Error(w, "", 500)
	}
}
