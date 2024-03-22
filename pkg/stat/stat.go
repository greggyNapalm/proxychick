package stat

import (
	"fmt"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/montanaflynn/stats"
	"github.com/oschwald/geoip2-golang"
	"io"
	"net"
	"strconv"
	"strings"
)

var defaultPercentiles = []float64{50.0, 75.0, 85.0, 90.0, 95.0, 99.0, 100.0}

type ProxyChickStatTable interface {
	createTable() table.Writer
	printTable()
	add(string)
}

type TableCountableRow struct {
	Value      string  `json:"value"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percent"`
}
type TableCountable struct {
	Name               string               `json:"name"`
	TableType          string               `json:"TableType"`
	Headers            table.Row            `json:"headers"`
	Rows               []*TableCountableRow `json:"rows"`
	outputs            []io.Writer          `json:"-"`
	TableWriter        table.Writer         `json:"-"`
	DistinctCntr       map[string]int       `json:"-"`
	TotalCnt           int                  `json:"-"`
	DistinctPerc       map[string]float64   `json:"-"`
	defaultPercentiles []float64            `json:"-"`
}

func NewTableCountable(tblName string, outputs []io.Writer) *TableCountable {
	var c TableCountable
	c.Name = tblName
	c.TableType = "countable"
	c.Headers = table.Row{"value", "count", "percent"}
	c.outputs = outputs
	c.DistinctCntr = make(map[string]int)
	return &c
}

func (self *TableCountable) add(cell string) {
	self.DistinctCntr[cell]++
	self.TotalCnt++
}

func (self *TableCountable) calcPerc() map[string]float64 {
	rv := make(map[string]float64)
	for k, v := range self.DistinctCntr {
		rv[k] = (float64(v) / float64(self.TotalCnt)) * 100
	}
	self.DistinctPerc = rv
	return rv
}

func (self *TableCountable) createTable() table.Writer {
	t := table.NewWriter()
	for _, o := range self.outputs {
		t.SetOutputMirror(o)
	}
	t.AppendHeader(self.Headers)
	self.calcPerc()
	for colName, colCnt := range self.DistinctCntr {
		t.AppendRow([]interface{}{colName, colCnt, fmt.Sprintf("%.2f", self.DistinctPerc[colName])})
		self.Rows = append(self.Rows, &TableCountableRow{colName, colCnt, self.DistinctPerc[colName]})
	}
	t.SortBy([]table.SortBy{
		{Name: "count", Mode: table.DscNumeric},
	})
	return t
}
func (self *TableCountable) printTable() {
	w := self.createTable()
	if len(self.outputs) > 0 {
		fmt.Println("\n", self.Name)
		w.Render()
	}
}

type ColumnMesurable struct {
	ColName     string             `json:"name"`
	vals        []float64          `json:"-"`
	Percentiles []float64          `json:"-"`
	Quantiles   map[string]float64 `json:"quantiles"`
}

func NewColumnMesurable(ColName string) *ColumnMesurable {
	var c ColumnMesurable
	c.ColName = ColName
	c.Percentiles = defaultPercentiles
	return &c
}

func (self *ColumnMesurable) calcPercentiles() map[string]float64 {
	rv := make(map[string]float64)
	for _, p := range self.Percentiles {
		quantile, _ := stats.Percentile(self.vals, p)
		rv[fmt.Sprintf("%.0f", p)] = quantile
	}
	self.Quantiles = rv
	return rv
}

// Container for mesurable results. Help to calc quantiles and organise results into table.
type TableMesurable struct {
	Name        string             `json:"name"`
	TableType   string             `json:"TableType"`
	Headers     table.Row          `json:"headers"`
	Rows        []*ColumnMesurable `json:"rows"`
	outputs     []io.Writer        `json:"-"`
	Percentiles []float64          `json:"-"`
	percentiles map[string]float64 `json:"percentiles"`
	Metrics     []*ColumnMesurable `json:"-"`
}

func NewTableMesurable(tblName string, outputs []io.Writer, metrics []*ColumnMesurable) *TableMesurable {
	var c TableMesurable
	c.Name = tblName
	c.TableType = "mesurable"
	c.Percentiles = defaultPercentiles
	header := table.Row{"name"}
	for _, pVal := range c.Percentiles {
		pName := fmt.Sprintf("%.0f", pVal)
		header = append(header, pName)
	}
	c.Headers = header
	c.outputs = outputs
	c.Metrics = metrics
	return &c
}

func (self *TableMesurable) add(s string) {
}

func (self *TableMesurable) createTable() table.Writer {
	t := table.NewWriter()
	for _, o := range self.outputs {
		t.SetOutputMirror(o)
	}
	t.AppendHeader(self.Headers)
	for _, m := range self.Metrics {
		self.Rows = append(self.Rows, m)
		row := table.Row{m.ColName}
		m.calcPercentiles()
		for _, pVal := range m.Percentiles {
			row = append(row, fmt.Sprintf("%.0f", m.Quantiles[fmt.Sprintf("%.0f", pVal)]))
		}
		t.AppendRow(row)
	}
	return t
}
func (self *TableMesurable) printTable() {
	w := self.createTable()
	if len(self.outputs) > 0 {
		fmt.Println("\n", self.Name)
		w.Render()
	}
}

type IPGeo struct {
	CountryName string
	CountryISO  string
}

func ProcTestResults(results []*client.Result, outputs []io.Writer, trasnport string) []ProxyChickStatTable {
	rv := []ProxyChickStatTable{}
	var colSucc, colErr, colTgtStatus, colPrxStatus, tblLatency ProxyChickStatTable
	var measurableMetrics []*ColumnMesurable
	var containsHTTPscheme = false
	colSucc = NewTableCountable("Success Rate", outputs)
	colErr = NewTableCountable("Errors", outputs)
	colTgtStatus = NewTableCountable("Taget HTTP status codes", outputs)
	colPrxStatus = NewTableCountable("Proxy HTTP status codes", outputs)

	latTTFB := NewColumnMesurable("TTFB")
	latDNS := NewColumnMesurable("DNS resolve")
	latConnect := NewColumnMesurable("Connect")
	latPrxResp := NewColumnMesurable("ProxyResp")
	latTLS := NewColumnMesurable("TLSHandshake")
	for _, r := range results {
		if strings.HasPrefix(r.ProxyURL.String(), "http") {
			containsHTTPscheme = true
		}
		if r.Status {
			colSucc.add("ok")
			colErr.add("ok")
			if trasnport == "tcp" {
				// these metrics collection implemented only for TCP and they will eq to 0(zero) in case of error
				latDNS.vals = append(latDNS.vals, float64(r.Latency.DNSresolve))
				latConnect.vals = append(latConnect.vals, float64(r.Latency.Connect))
				latTLS.vals = append(latTLS.vals, float64(r.Latency.TLSHandshake))
			}
			// these metrics works for both transport protocols TCP and UDP
			latTTFB.vals = append(latTTFB.vals, float64(r.Latency.TTFB))
			latPrxResp.vals = append(latPrxResp.vals, float64(r.Latency.ProxyResp))
		} else {
			colSucc.add("error")
			errNorm, _ := r.Error.MarshalCSV()
			colErr.add(errNorm)
		}
		if trasnport == "tcp" {
			colTgtStatus.add(strconv.Itoa(r.TargetStatusCode))
			colPrxStatus.add(strconv.Itoa(r.ProxyStatusCode))
		}
	}
	colSucc.printTable()
	colErr.printTable()
	rv = append(rv, colSucc, colErr)
	measurableMetrics = []*ColumnMesurable{latTTFB}
	if trasnport == "tcp" {
		measurableMetrics = append(measurableMetrics, latDNS, latConnect, latTLS)
		colTgtStatus.printTable()
		rv = append(rv, colTgtStatus)
		if containsHTTPscheme {
			colPrxStatus.printTable()
			rv = append(rv, colPrxStatus)
			measurableMetrics = append(measurableMetrics, latPrxResp)
		}
	}
	if trasnport == "udp" {
		measurableMetrics = append(measurableMetrics, latPrxResp)
	}
	tblLatency = NewTableMesurable("Latency", outputs, measurableMetrics)
	tblLatency.printTable()
	rv = append(rv, tblLatency)
	//return []ProxyChickStatTable{colSucc, colErr, colTgtStatus, colPrxStatus, tblLatency}
	return rv
}

func getCountyByIp(ipAddr net.IP, db geoip2.Reader) (IPGeo, error) {
	record, err := db.Country(ipAddr)
	//fmt.Println("record", record)
	if err != nil {
		return IPGeo{}, err
	}
	return IPGeo{record.Country.Names["en"], record.Country.IsoCode}, nil
}

func ProcIPTestResults(results []*client.Result, outputs []io.Writer, db geoip2.Reader) ([]ProxyChickStatTable, int) {
	countIPCountryTbl := NewTableCountable("Exit nodes country", outputs)
	uniqueIP := map[string]bool{}
	for _, r := range results {
		uniqueIP[r.ProxyNodeIPAddr.String()] = true
		geo, err := getCountyByIp(r.ProxyNodeIPAddr, db)
		//fmt.Println(geo, err)
		if err == nil {
			//fmt.Println("---")
			//fmt.Println(fmt.Sprintf("%s - %s", geo.CountryISO, geo.CountryName))
			//fmt.Println("---")
			countIPCountryTbl.add(fmt.Sprintf("%s - %s", geo.CountryISO, geo.CountryName))
		}
	}
	countIPCountryTbl.printTable()
	//fmt.Println(countIPCountryTbl)
	return []ProxyChickStatTable{countIPCountryTbl}, len(uniqueIP)
}
