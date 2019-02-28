// +build windows

//go:generate go get github.com/akavel/rsrc
//go:generate rsrc -ico selectbrowser.ico -o main.syso
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

var iePath string     //:= `C:\Program Files\Internet Explorer\IEXPLORE.EXE`
var chromePath string //:= `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`
var defaultbrowser string
var nondefaulthosts string

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func initialize() error {
	startmenuinternet, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Clients\StartMenuInternet`, registry.READ)
	if err != nil {
		return err
	}
	defer startmenuinternet.Close()

	chromeopenkey, err := registry.OpenKey(startmenuinternet, `Google Chrome\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer chromeopenkey.Close()

	chromePath, _, err = chromeopenkey.GetStringValue("")
	if err != nil {
		return err
	}
	chromePath = trimQuotes(chromePath)

	ieopenkey, err := registry.OpenKey(startmenuinternet, `IEXPLORE.EXE\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer ieopenkey.Close()
	iePath, _, err = ieopenkey.GetStringValue("")
	if err != nil {
		return err
	}
	iePath = trimQuotes(iePath)

	defaultbrowser = "IE"
	nondefaulthosts = "outlook.office.com,sharepoint.com,teams.microsoft.com,www.onenote.com,admin.microsoft.com"

	sburlappkey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Classes\SBUrl\Application`, registry.QUERY_VALUE)
	if err == nil {
		defaultbrowser, _, _ = sburlappkey.GetStringValue("DefaultBrowser")
		nondefaulthosts, _, _ = sburlappkey.GetStringValue("NonDefaultHosts")
	}

	return nil
}

func register() error {
	execpath, _ := os.Executable()

	classesroot, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Classes`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer classesroot.Close()

	sburlkey, _, err := registry.CreateKey(classesroot, `SBUrl`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer sburlkey.Close()

	sburlkey.SetStringValue("", "MaxOffice SelectBrowser URL Handler")

	sburlappkey, _, err := registry.CreateKey(sburlkey, `Application`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer sburlappkey.Close()

	sburlappkey.SetStringValue("ApplicationName", "MaxOffice SelectBrowser")
	sburlappkey.SetStringValue("ApplicationDescription", "Open specified sites in IE or Chrome")
	sburlappkey.SetStringValue("DefaultBrowser", "IE")
	sburlappkey.SetStringValue("NonDefaultHosts", "www.office.com,outlook.office.com,sharepoint.com,teams.microsoft.com,www.onenote.com,admin.microsoft.com")

	sburlcommandkey, _, err := registry.CreateKey(sburlkey, `shell\open\command`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer sburlcommandkey.Close()

	sburlcommandkey.SetStringValue("", execpath+" %1")

	startmenuinternet, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Clients\StartMenuInternet`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer startmenuinternet.Close()

	selectbrowserkey, _, err := registry.CreateKey(startmenuinternet, "selectbrowser", registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer selectbrowserkey.Close()

	selectbrowserkey.SetStringValue("", "MaxOffice SelectBrowser")

	capskey, _, err := registry.CreateKey(selectbrowserkey, `Capabilities\UrlAssociations`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer capskey.Close()

	capskey.SetStringValue("http", "SBUrl")
	capskey.SetStringValue("https", "SBUrl")

	registeredappkey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\RegisteredApplications`, registry.ALL_ACCESS)
	if err != nil {
		log.Println("RegisteredApplications")
		return err
	}
	defer registeredappkey.Close()

	registeredappkey.SetStringValue("selectbrowser", `Software\Clients\StartMenuInternet\selectbrowser\Capabilities`)

	return nil
}

func unregister() error {
	registeredappkey, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\RegisteredApplications`, registry.ALL_ACCESS)
	if err != nil {
		log.Println("RegisteredApplications")
		return err
	}
	defer registeredappkey.Close()

	registeredappkey.DeleteValue("selectbrowser")

	classesroot, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Classes`, registry.ALL_ACCESS)
	if err != nil {
		log.Println("Classes")
		return err
	}
	defer classesroot.Close()

	err = registry.DeleteKey(classesroot, `SBUrl\shell\open\command`)
	err = registry.DeleteKey(classesroot, `SBUrl\shell\open`)
	err = registry.DeleteKey(classesroot, `SBUrl\shell`)
	err = registry.DeleteKey(classesroot, `SBUrl\Application`)
	err = registry.DeleteKey(classesroot, `SBUrl`)

	startmenuinternet, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Clients\StartMenuInternet`, registry.ALL_ACCESS)
	if err != nil {
		log.Println("StartmenuInternet")
		return err
	}
	defer startmenuinternet.Close()

	err = registry.DeleteKey(startmenuinternet, `selectbrowser\Capabilities\UrlAssociations`)
	err = registry.DeleteKey(startmenuinternet, `selectbrowser\Capabilities`)
	err = registry.DeleteKey(startmenuinternet, `selectbrowser`)

	if err != nil {
		log.Println("selectbrowser error")
		return err
	}

	return nil
}

func invokeChrome(urlToInvoke string) {
	browser := exec.Command(chromePath, "--", urlToInvoke)
	err := browser.Start()
	if err != nil {
		log.Printf("Error invoking Chrome:%v\n", err)
	}
}

func invokeIE(urlToInvoke string) {
	browser := exec.Command(iePath, urlToInvoke)
	err := browser.Start()
	if err != nil {
		log.Printf("Error invoking IE:%v\n", err)
	}
}

func invokeBrowser(urlToInvoke string) {
	fullURL, err := url.Parse(urlToInvoke)
	if err != nil {
		log.Fatal("Invalid url:" + urlToInvoke)
	}

	invokedHost := strings.ToLower(fullURL.Hostname())
	nondefault := strings.Contains(nondefaulthosts, invokedHost)

	if defaultbrowser != "IE" {
		nondefault = !nondefault
	}

	if nondefault {
		log.Printf("Invoking Chrome for url:" + urlToInvoke)
		invokeChrome(urlToInvoke)
	} else {
		log.Printf("Invoking IE for url:" + urlToInvoke)
		invokeIE(urlToInvoke)
	}
}

func main() {
	var err error

	today := time.Now()
	logfilename := path.Join(os.Getenv("TEMP"), fmt.Sprintf("selectbrowser-%d-%02d-%02d.log", today.Year(), today.Month(), today.Day()))
	file, err := os.OpenFile(logfilename, os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	log.SetOutput(file)

	var registerflag = flag.Bool("register", false, "Register MaxOffice Select Browser")
	var unregisterflag = flag.Bool("unregister", false, "Unregister MaxOffice Select Browser")

	flag.Parse()

	switch {
	case *registerflag:
		err = register()
		if err != nil {
			log.Fatal(err)
		}
	case *unregisterflag:
		err = unregister()
		if err != nil {
			log.Fatal(err)
		}
	default:
		if len(os.Args) > 1 {
			err = initialize()
			if err != nil {
				log.Fatal(err)
			}

			var args = os.Args[1:]
			invokeBrowser(args[0])
		}
	}
}
