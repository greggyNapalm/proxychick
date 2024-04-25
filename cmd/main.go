package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"github.com/greggyNapalm/proxychick/pkg/job"
	"github.com/greggyNapalm/proxychick/pkg/stat"
	"github.com/greggyNapalm/proxychick/pkg/utils"
	"github.com/oschwald/geoip2-golang"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
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
	timeOut             time.Duration
	loop                int
	transport           string
	countryMmdbPath     string
}

func NewCmdCfg() CmdCfg {
	defaultTCPTarget := "https://api.datascrape.tech/latest/ip"
	defaultUDPTarget := "udp://api.datascrape.tech:80"
	var rv = CmdCfg{}
	var err error
	flag.IntVar(&rv.maxConcurrency, "c", 300, "number of simultaneous HTTP requests(maxConcurrency)")
	flag.StringVar(&rv.inPath, "i", "proxylist.txt", "path to the proxylist file or STDIN")
	flag.StringVar(&rv.outPath, "o", "STDOUT", "path to the results file")
	flag.StringVar(&rv.prxProto, "p", "http", "Proxy protocol. If not specified in proxy URL, choose one of http/https/socks4/socks4a/socks5/socks5h")
	var timeOut = flag.String("to", "10s", "Timeout for entire request")
	flag.IntVar(&rv.loop, "loop", 1, "Loop over proxylist content N times")
	flag.StringVar(&rv.transport, "transport", "tcp", "Transport protocol for interaction with the target. Will be incapsulated into proxy protocol.")
	var pBarDisabled = flag.Bool("noProgresBar", false, "Disable the progress meter")
	var statDisabled = flag.Bool("noStat", false, "Disable stats output")
	var targetAddr = flag.String("t", defaultTCPTarget, "Target URL(TCP) and HOST:PORT(UDP)")
	var showVersion = flag.Bool("version", false, "Show version and exit")
	var debugCmd = flag.Bool("verbose", false, "Enables debug logs")
	flag.StringVar(&rv.countryMmdbPath, "countryMmdb", "", "Path to GeoLite2-Country.mmdb")
	flag.Parse()

	rv.timeOut, err = time.ParseDuration(*timeOut)
	if err != nil {
		log.Fatal("Can't parse timeout(to) cmd param:" + err.Error())
	}
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
		if *targetAddr == defaultTCPTarget {
			rv.targetAddr = defaultUDPTarget
		} else {
			rv.targetAddr = *targetAddr
		}
		rv.prxProto = "socks5"
		targetURL, err := url.Parse(rv.targetAddr)
		if err != nil {
			log.Fatal("Can't parse Target URL:" + rv.targetAddr)
			panic(err)
		}
		rv.targetURL = targetURL
	}
	if rv.countryMmdbPath == "" {
		countryMmdbPathEnv := os.Getenv("PROXYCHICK_MMDB_COUNTRY")
		if countryMmdbPathEnv != "" {
			rv.countryMmdbPath = countryMmdbPathEnv
		}
	}
	if rv.countryMmdbPath != "" {
		_, err := geoip2.Open(rv.countryMmdbPath)
		if err != nil {
			log.Fatal("Failed to open GeoLite2-Country.mmdb - ", err)
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
		bytes, err = os.ReadFile(inPath)
	}
	if err != nil {
		log.Fatal("Can't read file:" + inPath)
	}
	for _, el := range strings.Split(string(bytes), "\n") {
		if el != "" {
			rv = append(rv, strings.Replace(el, "\r", "", -1))
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
		rv, err = gocsv.MarshalString(results)
	}
	if format == "json" {
		var jsonDoc []byte
		//TODO: impl working JSON serialisation
		//fmt.Printf("t1: %T\n", results[0])
		jsonDoc, err = json.Marshal(results)
		rv = string(jsonDoc)
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
	jobMetrics := job.JobMetrics{}
	var results []*client.Result
	var bar *progressbar.ProgressBar
	var pStringsRaw []string
	var pStringsFormated []*url.URL
	statOutputs := []io.Writer{os.Stdout}
	cmdCfg := NewCmdCfg()
	pStringsRaw = GetProxyStrings(cmdCfg.inPath)
	resultsCh := make(chan client.Result, len(pStringsRaw))
	jobCfg := job.PListEvanJobCfg{
		cmdCfg.maxConcurrency,
		*cmdCfg.targetURL,
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
	JobStarted := time.Now()
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
	jobMetrics.Duration = time.Since(JobStarted)
	outTxt, _ := formatResulst(results, "csv")
	retFinalText(cmdCfg.outPath, outTxt)
	if cmdCfg.isStatsEnables {
		_ = stat.ProcTestResults(results, statOutputs, cmdCfg.transport, &jobMetrics)

		if cmdCfg.countryMmdbPath != "" {
			db, err := geoip2.Open(cmdCfg.countryMmdbPath)
			defer db.Close()
			if err == nil {
				_ = stat.ProcIPTestResults(results, statOutputs, *db)

			}
		}
	}
	fmt.Println("Duration:", fmt.Sprintf("%s", jobMetrics.Duration))
	fmt.Println("Unique Exit Nodes IPs:", jobMetrics.UniqueExitNodesIPCnt,
		fmt.Sprintf(" (%.0f%% of Rquests and ", 100.00*float64(jobMetrics.UniqueExitNodesIPCnt)/float64(jobMetrics.ReqsCnt)),
		fmt.Sprintf("%.0f%% of Responces)", 100.00*float64(jobMetrics.UniqueExitNodesIPCnt)/float64(jobMetrics.RespCnt)),
	)
}
