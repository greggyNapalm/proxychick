package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/greggyNapalm/proxychick/internal/httpx"
	"github.com/greggyNapalm/proxychick/internal/job"
	"github.com/greggyNapalm/proxychick/internal/utils"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
)

type CmdCfg struct {
	maxConcurrency int
	targetURL      *url.URL
	inPath         string
	outPath        string
	isSilent       bool
	prxProto       string
	timeOut        int
}

func NewCmdCfg() CmdCfg {
	var rv = CmdCfg{}
	flag.IntVar(&rv.maxConcurrency, "c", 300, "number of simultaneous HTTP requests(maxConcurrency)")
	flag.StringVar(&rv.inPath, "i", "STDIN", "path to the proxylist file")
	flag.StringVar(&rv.outPath, "o", "STDOUT", "path to the results file")
	flag.BoolVar(&rv.isSilent, "s", false, "Disable the progress meter")
	flag.StringVar(&rv.prxProto, "p", "http", "Proxy protocol. If not specified in list, choose one of http/https/socks4/socks4a/socks5/socks5h")
	flag.IntVar(&rv.timeOut, "to", 10, "Timeout for entire HTTP request in seconds")

	var targetURLStr = flag.String("t", "https://www.cloudflare.com/cdn-cgi/trace", "Target URL")
	flag.Parse()

	targetURL, err := url.Parse(*targetURLStr)
	if err != nil {
		log.Fatal("Can't parse Target URL:" + *targetURLStr)
		panic(err)
	}
	rv.targetURL = targetURL
	return rv
}

func GetProxyStrings(inPath string) []string {
	rv := []string{}
	var bytes []byte
	var err error
	if inPath == "STDIN" {
		bytes, err = io.ReadAll(os.Stdin)
	} else {
		bytes, err = ioutil.ReadFile(inPath)
	}
	if err != nil {
		log.Fatal("Can't read file:" + inPath)
	}
	for _, el := range strings.Split(string(bytes), "\n") {
		if el != "" {
			rv = append(rv, el)
		}
	}
	return rv
}

func formatResulst(results []*httpx.Result, format string) (rv string, err error) {
	if format == "" {
		format = "csv"
	}

	err = errors.New("formatResuls: unsuported result format " + format)
	if format == "csv" {
		rv, err = gocsv.MarshalString(&results)
	}
	if format == "json" {
		//TODO: impl working JSON serialisation
		fmt.Printf("t1: %T\n", results[0])
		//rv, err = json.Marshal(results[0])
	}
	return
}

func retFinalText(outPath string, txt string) {
	if outPath == "STDOUT" {
		fmt.Print(txt)
	} else {
		utils.SaveSonDisk(outPath, txt)
	}
}

func main() {
	var results []*httpx.Result
	var bar *progressbar.ProgressBar
	var pStringsRaw []string
	var pStringsFormated []*url.URL
	cmdCfg := NewCmdCfg()
	pStringsRaw = GetProxyStrings(cmdCfg.inPath)
	//fmt.Printf("%+q\n", pStringsRaw)
	resultsCh := make(chan httpx.Result, len(pStringsRaw))
	jobCfg := job.PListEvanJobCfg{
		cmdCfg.maxConcurrency,
		*cmdCfg.targetURL,
		cmdCfg.timeOut,
	}
	for _, PrxStrRaw := range pStringsRaw {
		prxURL, err := job.AdaptRowProxyStr(PrxStrRaw, cmdCfg.prxProto)
		if err != nil {
			log.Fatal(PrxStrRaw + " | " + err.Error())
		} else {
			pStringsFormated = append(pStringsFormated, prxURL)
		}
	}
	go job.EvaluateProxyList(pStringsFormated, &jobCfg, resultsCh)
	if !(cmdCfg.isSilent) {
		bar = progressbar.Default(int64(len(pStringsFormated)))
	}
	for i := 0; i < len(pStringsFormated); i++ {
		res := <-resultsCh
		results = append(results, &res)
		if !(cmdCfg.isSilent) {
			bar.Add(1)
		}
	}
	outTxt, _ := formatResulst(results, "csv")
	retFinalText(cmdCfg.outPath, outTxt)
}
