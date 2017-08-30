package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

type domain struct {
	name      string
	zipData   *bytes.Buffer
	zipWriter *zip.Writer
	textInZip io.Writer
	mutex     sync.Mutex
	nr        int64
	fileNr    int
	lastFlush time.Time
}

// newDomain returns a new *domain with domain.Name set to name
func newDomain(name string) *domain {
	d := new(domain)
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.zipData = new(bytes.Buffer)
	d.name = name
	// Create a new zip archive.
	d.zipWriter = zip.NewWriter(d.zipData)

	f, err := d.zipWriter.Create(d.name + ".txt")
	if err != nil {
		log.Fatal(err)
	}

	d.textInZip = f

	go d.resetFileNr()

	return d
}

func (d *domain) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ref, err := url.Parse(req.Referer())
	if err != nil {
		http.Error(w, "", 500)
		return
	}

	if ref.Host == globalConfig.ZipPageURI {
		d.flush()
		http.Redirect(w, req, req.Referer(), 303)
	} else {
		http.Error(w, "", 500)
	}
}

// resetFileNr resets d.fileNr every day at 00:00.
// Note that resetFileNr will not return so call it in a new gorutine.
func (d *domain) resetFileNr() {
	newDay := time.Now().Add(time.Hour * 24).Truncate(time.Hour * 24)
	time.Sleep(time.Until(newDay))

	ticker := time.NewTicker(time.Hour * 24)
	for range ticker.C {
		d.mutex.Lock()
		d.fileNr = 0
		if time.Since(d.lastFlush) > time.Hour*24*30 {
			d.flush()
		}
		d.mutex.Unlock()
	}
}

// flush writes all csp reports related to d to the current .zip file, if no .zip file exsists then a new .zip file is created
func (d *domain) flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.nr > 0 {
		// Close Zip file
		err := d.zipWriter.Close()
		if err != nil {
			log.Println("ZipWriter.Close() :", err)
			return
		}
		// Zip file name as: example.com_YYYY-MMM-DD_i.zip
		zipName := globalConfig.ZipsDir + d.name + "_" + time.Now().Format("2006-01-02") + "_" + strconv.Itoa(d.fileNr) + ".zip"
		for _, err = os.Stat(zipName); !os.IsNotExist(err); d.fileNr++ {
			zipName = globalConfig.ZipsDir + d.name + "_" + time.Now().Format("2006-01-02") + "_" + strconv.Itoa(d.fileNr) + ".zip"
			_, err = os.Stat(zipName)
		}

		err = ioutil.WriteFile(zipName, d.zipData.Bytes(), 0644)
		if err != nil {
			log.Println("ioutil.WriteFile() :", err)
			return
		}
		// Empty the buffer
		d.zipData.Reset()

		d.zipWriter = zip.NewWriter(d.zipData)

		// Create file in zip archive.
		f, err := d.zipWriter.Create(d.name + ".txt")
		if err != nil {
			log.Println("ZipWriter.Create() :", err)
		}
		d.textInZip = f
		d.nr = 0
		d.lastFlush = time.Now()
	}
}
