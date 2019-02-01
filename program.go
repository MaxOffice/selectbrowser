package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"golang.org/x/sys/windows/registry"
)

var iePath string     //:= `C:\Program Files\Internet Explorer\IEXPLORE.EXE`
var chromePath string //:= `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`

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
		// Report error somehow
		return err
	}
	defer startmenuinternet.Close()

	chromeopenkey, err := registry.OpenKey(startmenuinternet, `Google Chrome\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		// Report error somehow
		return err
	}
	defer chromeopenkey.Close()
	chromePath, _, err = chromeopenkey.GetStringValue("")
	if err != nil {
		// Report error somehow
		return err
	}
	chromePath = trimQuotes(chromePath)

	ieopenkey, err := registry.OpenKey(startmenuinternet, `IEXPLORE.EXE\shell\open\command`, registry.QUERY_VALUE)
	if err != nil {
		// Report error somehow
		return err
	}
	defer ieopenkey.Close()
	iePath, _, err = ieopenkey.GetStringValue("")
	if err != nil {
		// Report error somehow
		return err
	}
	iePath = trimQuotes(iePath)

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

	sburlkey.SetStringValue("", "MaxOffice Select Browser URL Handler")

	sburlappkey, _, err := registry.CreateKey(sburlkey, `Application`, registry.ALL_ACCESS)
	if err != nil {
		return err
	}
	defer sburlappkey.Close()

	sburlappkey.SetStringValue("ApplicationName", "MaxOffice Select Browser")
	sburlappkey.SetStringValue("ApplicationDescription", "Open specified sites in IE or Chrome")

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

	selectbrowserkey.SetStringValue("", "MaxOffice Select Browser")

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

func invokeChrome(url string) {
	browser := exec.Command(chromePath, "--", url)
	err := browser.Start()
	if err != nil {
		log.Printf("Error invoking Chrome:%v\n", err)
	}
}

func invokeIE(url string) {
	browser := exec.Command(iePath, url)
	err := browser.Start()
	if err != nil {
		log.Printf("Error invoking IE:%v\n", err)
	}
}

func invokeBrowser(url string) {
	if url == "http://www.microsoft.com/" || url == "http://www.azure.com/" {
		fmt.Println("Invoking IE")
		invokeIE(url)
	} else {
		fmt.Println("Invoking Chrome")
		invokeChrome(url)
	}
}

func main() {
	var err error
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
			/*
				log.Printf("IE:%v  Chrome:%v\n", iePath, chromePath)

				fmt.Println("Some arguments were passed. These are:")

				for i := range args {
					log.Println(args[i])
				}
			*/

			invokeBrowser(args[0])
		}
		log.Println("Selection complete.")
	}
}
