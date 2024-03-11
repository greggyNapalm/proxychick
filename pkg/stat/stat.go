package stat

import (
	"fmt"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/montanaflynn/stats"
	"io"
	"strconv"
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
	c.Headers = table.Row{"Value", "Count", "Percent"}
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
		{Name: "Count", Mode: table.Dsc},
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
	outputs     []io.Writer
	Percentiles []float64
	percentiles map[string]float64
	Metrics     []*ColumnMesurable
}

func NewTableMesurable(tblName string, outputs []io.Writer, metrics []*ColumnMesurable) *TableMesurable {
	var c TableMesurable
	c.Name = tblName
	c.TableType = "mesurable"
	c.Percentiles = defaultPercentiles
	header := table.Row{"Name"}
	for _, pVal := range c.Percentiles {
		pName := fmt.Sprintf("%.0f", pVal)
		header = append(header, pName+"%p")
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

func ProcTestResults(results []*client.Result, outputs []io.Writer, trasnport string) []ProxyChickStatTable {
	var colSucc, colErr, colTgtStatus, colPrxStatus, tblLatency ProxyChickStatTable
	var measurableMetrics []*ColumnMesurable
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
	if trasnport == "tcp" {
		colTgtStatus.printTable()

		colPrxStatus.printTable()
		measurableMetrics = []*ColumnMesurable{latDNS, latTTFB, latConnect, latPrxResp, latTLS}
	} else {
		measurableMetrics = []*ColumnMesurable{latPrxResp, latTTFB}
	}
	tblLatency = NewTableMesurable("Latency", outputs, measurableMetrics)
	tblLatency.printTable()
	return []ProxyChickStatTable{colSucc, colErr, colTgtStatus, colPrxStatus, tblLatency}
	//jsonDoc, _ := json.MarshalIndent([]ProxyChickStatTable{colSucc, colErr, colTgtStatus, colPrxStatus, tblLatency}, "", "    ")
	//fmt.Println(string(jsonDoc))
}
