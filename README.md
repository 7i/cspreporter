# cspreporter
cspreporter listens to CSP reports from whitelisted sites and creates zip files for easy access to the reports.

cspreporter is designed to be used behind a SSL terminator like F5 or NGINX to allow CSP reports to be sent encrypted.

cspreporter will read cspreporter.conf and set the following parameters:
ZipPageURI - Internal page used by developers of domains whitelisted in DomainsWhitelist for downloading CSP reports.
ReportURI - Externally accessible DNS adress and port to this CSP report server ( e.g. csp.example.com:8080 ) 
DomainsWhitelist - List of whitelisted domains that send their CSP reports to this report server
Syslog - Send CSP reports to syslog server 
Transport - Use tcp or udp for syslog packages 
MaxReportsPerZip - Maximum number of reports saved in a zip before it is automaticly saved to disk
ZipsDir - Directory to save all zip files containing all CSP reports
TemplateDir - Directory to find index.tmpl, domain.tmpl and csp.tmpl
MaxCSPReportSize - Maximum size in bytes for one CSP report ( http.MaxBytesReader(w, req.Body, MaxCSPReportSize) )
Silent - Suppress all command line output

Checklist:
Configure cspreporter.conf with the parameters that match your needs (note that the config needs to follow JSON format)
Set up ZipPageURI to only be accessible on the internal network
Set up ReportURI to match the servers DNS name and port
Set up the sites listed in DomainsWhitelist to use the same value as ReportURI in their report-uri parameter in their CSP headers

Usage: 
cspreporter -conf /path/to/cspreporter.conf (default "./cspreporter.conf if no parameter is used)
