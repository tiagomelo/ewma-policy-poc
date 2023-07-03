package parser

import (
	"net"
	"os"
	"text/template"

	"github.com/pkg/errors"
)

type Data struct {
	IP   string
	Port int
}

func ParseWithLocalIpAddr(serverPort int, templateFile, outputFile string) error {
	ip, err := getOutboundIpAddr()
	if err != nil {
		return errors.Wrap(err, "getting ip")
	}
	data := &Data{
		IP:   ip,
		Port: serverPort,
	}
	tmpl, err := template.ParseFiles(templateFile)
	if err != nil {
		return errors.Wrapf(err, `parsing template file "%s"`, templateFile)
	}
	file, err := os.Create(outputFile)
	if err != nil {
		return errors.Wrapf(err, `creating output file "%s"`, outputFile)
	}
	defer file.Close()
	if err = tmpl.Execute(file, data); err != nil {
		return errors.Wrapf(err, `executing template file "%s"`, templateFile)
	}
	return nil
}

func getOutboundIpAddr() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
