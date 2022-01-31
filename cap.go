package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type TlsClientCert struct {
	Serial string
}
type Context_info struct {
	MacAddr       string
	AcctSessionId string
	DeviceType    string
	ClientCert    TlsClientCert
	TimeStamp     uint64
	IsActive      bool
	DeviceName    string
}

func loadCAPTemplate(name string) *template.Template {
	t, err := template.ParseFiles(
		"front/"+name+".gtpl",
		"front/components/header.gtpl",
		"front/components/footer.gtpl",
	)
	if err != nil {
		log.Fatalf("template error: %v", err)
	}
	return t
}

var registered = make(map[string]Context_info)
var unlinked = make(map[string]Context_info)
var macaddrToSerial = make(map[string]string)

func setFromAuthdetail(authRow string) {
	auths := strings.Split(authRow, "\n\n")
	for _, authEach := range auths {
		detail := strings.Split(authEach, "\n")
		for i := range detail {
			detail[i] = strings.TrimSpace(detail[i])
		}
		authMap := make(map[string]string)
		for _, v := range detail[1:] {
			// all target strings are "xxx = yyy"
			splited := strings.Split(v, "=")
			front := strings.TrimSpace(splited[0])
			back := strings.TrimSpace(splited[1])
			authMap[front] = back
		}

		serial := authMap["TLS-Client-Cert-Serial"]
		var clientCert TlsClientCert
		clientCert.Serial = serial
		macaddr := authMap["Calling-Station-Id"]
		var contextInfo Context_info
		contextInfo.MacAddr = macaddr
		contextInfo.ClientCert = clientCert
		contextInfo.TimeStamp, _ = strconv.ParseUint(authMap["Timestamp"], 10, 64)
		contextInfo.IsActive = false

		if _, ok := registered[serial]; ok {
			registered[serial] = contextInfo
		} else {
			unlinked[serial] = contextInfo
		}
		macaddrToSerial[macaddr] = serial
	}
}
func getContext(serial string) (Context_info, bool) {
	if val, ok := registered[serial]; ok {
		return val, ok
	} else if val, ok := unlinked[serial]; ok {
		return val, ok
	}
	return Context_info{}, false
}
func setContext(serial string, context Context_info) {
	if _, ok := registered[serial]; ok {
		registered[serial] = context
	} else if _, ok := unlinked[serial]; ok {
		unlinked[serial] = context
	}
}
func UpdateContext(detailRow string, authRow string) {
	setFromAuthdetail(authRow)

	details := strings.Split(detailRow, "\n\n")
	for _, detailEach := range details {
		detail := strings.Split(detailEach, "\n")
		for i := range detail {
			detail[i] = strings.TrimSpace(detail[i])
		}
		detailMap := make(map[string]string)
		for _, v := range detail[1:] {
			// all target strings are "xxx = yyy"
			splited := strings.Split(v, "=")
			front := strings.TrimSpace(splited[0])
			back := strings.TrimSpace(splited[1])
			detailMap[front] = back
		}
		macaddr := detailMap["Calling-Station-Id"]
		serial, ok := macaddrToSerial[macaddr]
		if !ok {
			println("Not Found serial.")
			return
		}
		if detailMap["Acct-Status-Type"] == "Start" {
			context, ok := getContext(serial)
			if !ok {
				println("Not Found context")
				return
			}
			context.IsActive = true
			context.TimeStamp, _ = strconv.ParseUint(detailMap["Timestamp"], 10, 64)
			context.AcctSessionId = detailMap["Acct-Session-Id"]
			setContext(serial, context)
		} else if detailMap["Acct-Status-Type"] == "Interim-Update" {
			context, ok := getContext(serial)
			if !ok {
				println("Not Found context")
				return
			}
			context.TimeStamp, _ = strconv.ParseUint(detailMap["Timestamp"], 10, 64)
			setContext(serial, context)
		} else if detailMap["Acct-Status-Type"] == "Stop" {
			context, ok := getContext(serial)
			if !ok {
				println("Not Found context")
				return
			}
			context.IsActive = false
			context.TimeStamp, _ = strconv.ParseUint(detailMap["Timestamp"], 10, 64)
			context.AcctSessionId = detailMap["Acct-Session-Id"]
			context.MacAddr = ""
			delete(macaddrToSerial, macaddr)
			setContext(serial, context)
		}
	}
}

func CAP() {
	e := echo.New()
	e.Use(middleware.Logger())

	e.Static("/static/css/", "front/css")
	e.GET("/register", func(c echo.Context) error {
		clientCert := c.Request().TLS.PeerCertificates[0]
		serial := "\"" + fmt.Sprintf("%x", clientCert.SerialNumber) + "\""
		name := ""
		if val, ok := unlinked[serial]; ok {
			delete(unlinked, serial)
			registered[serial] = val
		} else if val, ok := registered[serial]; ok {
			name = val.DeviceName
		} else {
			var context Context_info
			context.IsActive = false
			context.ClientCert.Serial = serial
			registered[serial] = context
		}
		loadCAPTemplate("register_device").Execute(c.Response(), map[string]string{
			"Name":  name,
			"Title": "CAP",
		})
		return nil
	})
	e.POST("/register_name", func(c echo.Context) error {
		clientCert := c.Request().TLS.PeerCertificates[0]
		serial := "\"" + fmt.Sprintf("%x", clientCert.SerialNumber) + "\""
		name := c.FormValue("device_name")
		println(name, serial)
		if val, ok := registered[serial]; ok {
			val.DeviceName = name
			registered[serial] = val
		}
		loadCAPTemplate("registered").Execute(c.Response(), map[string]string{
			"Title": "CAP",
		})
		return nil
	})
	e.GET("/admin", func(c echo.Context) error {
		c.JSON(http.StatusAccepted, map[string]interface{}{
			"Unlinked":  unlinked,
			"Registerd": registered,
		})
		return nil
	})

	// Create a CA certificate pool and add cert.pem to it
	caCert, err := ioutil.ReadFile("secom_rootca.cer")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}

	// Create the TLS Config with the CA pool and enable Client certificate validation
	tlsConfig := &tls.Config{
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}
	// tlsConfig.BuildNameToCertificate()

	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}
	e.Logger.Fatal(e.StartServer(server))
}
