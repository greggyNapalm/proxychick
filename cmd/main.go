package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"github.com/greggyNapalm/proxychick/pkg/job"
	"github.com/greggyNapalm/proxychick/pkg/stat"
	"github.com/greggyNapalm/proxychick/pkg/utils"
	"github.com/schollz/progressbar/v3"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strings"
	"syscall"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	debug   = false
)

type CmdCfg struct {
	maxConcurrency      int
	targetURL           *url.URL
	targetAddr          string
	inPath              string
	outPath             string
	isPorgresBarEnabled bool
	isStatsEnables      bool
	prxProto            string
	timeOut             int
	loop                int
	transport           string
}

func NewCmdCfg() CmdCfg {
	defaultTCPTarget := "https://api.datascrape.tech/latest/ip"
	defaultUDPTarget := "api.datascrape.tech:80"
	var rv = CmdCfg{}
	flag.IntVar(&rv.maxConcurrency, "c", 300, "number of simultaneous HTTP requests(maxConcurrency)")
	flag.StringVar(&rv.inPath, "i", "proxylist.txt", "path to the proxylist file or STDIN")
	flag.StringVar(&rv.outPath, "o", "STDOUT", "path to the results file")
	flag.StringVar(&rv.prxProto, "p", "http", "Proxy protocol. If not specified in proxy URL, choose one of http/https/socks4/socks4a/socks5/socks5h")
	flag.IntVar(&rv.timeOut, "to", 10, "Timeout for entire HTTP request in seconds")
	flag.IntVar(&rv.loop, "loop", 1, "Loop over proxylist content N times")
	flag.StringVar(&rv.transport, "transport", "tcp", "Transport protocol for interaction with the target. Will be incapsulated into proxy protocol.")
	var pBarDisabled = flag.Bool("noProgresBar", false, "Disable the progress meter")
	var statDisabled = flag.Bool("noStat", false, "Disable stats output")
	var targetAddr = flag.String("t", defaultTCPTarget, "Target URL(TCP) and HOST:PORT(UDP)")
	var showVersion = flag.Bool("version", false, "Show version and exit")
	var debugCmd = flag.Bool("verbose", false, "Enables debug logs")
	flag.Parse()

	rv.isPorgresBarEnabled = !(*pBarDisabled)
	rv.isStatsEnables = !(*statDisabled)
	var debugEnv = os.Getenv("PROXYCHICK_DEBUG")
	if *showVersion {
		fmt.Printf("proxychick %s, commit %s, built at %s", version, commit, date)
		syscall.Exit(0)
	}
	if *debugCmd || debugEnv != "" {
		debug = true
	}
	if rv.transport == "tcp" {
		targetURL, err := url.Parse(*targetAddr)
		if err != nil {
			log.Fatal("Can't parse Target URL:" + *targetAddr)
			panic(err)
		}
		rv.targetURL = targetURL
	} else if rv.transport == "udp" {
		rv.targetURL = &url.URL{}
		if *targetAddr == defaultTCPTarget {
			rv.targetAddr = defaultUDPTarget
		} else {
			rv.targetAddr = *targetAddr
		}
	}

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

func formatResulst(results []*client.Result, format string) (rv string, err error) {
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
	var results []*client.Result
	var bar *progressbar.ProgressBar
	var pStringsRaw []string
	var pStringsFormated []*url.URL
	cmdCfg := NewCmdCfg()
	pStringsRaw = GetProxyStrings(cmdCfg.inPath)
	resultsCh := make(chan client.Result, len(pStringsRaw))
	jobCfg := job.PListEvanJobCfg{
		cmdCfg.maxConcurrency,
		*cmdCfg.targetURL,
		cmdCfg.targetAddr,
		cmdCfg.timeOut,
		cmdCfg.transport,
		debug,
	}
	for _, PrxStrRaw := range pStringsRaw {
		prxURL, err := job.AdaptRawProxyStr(PrxStrRaw, cmdCfg.prxProto)
		if err != nil {
			log.Fatal(PrxStrRaw + " | " + err.Error())
		} else {
			pStringsFormated = append(pStringsFormated, prxURL)
		}
	}
	if cmdCfg.loop > 1 {
		var tmpStringsFormated []*url.URL
		for _ = range cmdCfg.loop {
			tmpStringsFormated = append(tmpStringsFormated, pStringsFormated...)
		}
		pStringsFormated = tmpStringsFormated
	}
	go job.EvaluateProxyList(pStringsFormated, &jobCfg, resultsCh)
	if cmdCfg.isPorgresBarEnabled {
		bar = progressbar.Default(int64(len(pStringsFormated)))
	}
	for i := 0; i < len(pStringsFormated); i++ {
		res := <-resultsCh
		results = append(results, &res)
		if cmdCfg.isPorgresBarEnabled {
			bar.Add(1)
		}
	}
	outTxt, _ := formatResulst(results, "csv")
	retFinalText(cmdCfg.outPath, outTxt)
	if cmdCfg.isStatsEnables {
		stat.ProcTestResults(results)
	}
}
