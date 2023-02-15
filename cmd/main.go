package main

import (
	"flag"
	"fmt"
	"os"

	"k8s.io/klog/v2"
	"sigs.k8s.io/aws-file-cache-csi-driver/pkg/driver"
)

func main() {
	var (
		endpoint = flag.String("endpoint", "unix://tmp/csi.sock", "CSI Endpoint")
		version  = flag.Bool("version", false, "Print the version and exit")
	)
	klog.InitFlags(nil)
	flag.Parse()

	if *version {
		info, err := driver.GetVersionJSON()
		if err != nil {
			klog.Fatalln(err)
		}
		fmt.Println(info)
		os.Exit(0)
	}

	drv := driver.NewDriver(*endpoint)
	if err := drv.Run(); err != nil {
		klog.Fatalln(err)
	}
}
