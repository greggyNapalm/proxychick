package stat

import (
	"fmt"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/montanaflynn/stats"
	"os"
	"strconv"
)

var defaultPercentiles = []float64{50.0, 75.0, 85.0, 90.0, 95.0, 99.0, 100.0}

// Container for countable results. Help to calc percantage and organise results into table.
type ColumnStat struct {
	tblName            string
	DistinctCntr       map[string]int
	TotalCnt           int
	DistinctPerc       map[string]float64
	defaultPercentiles []float64
}

func NewColumnStat(tblName string) *ColumnStat {
	var c ColumnStat
	c.tblName = tblName
	c.DistinctCntr = make(map[string]int)
	c.defaultPercentiles = []float64{50.0, 75.0, 85.0, 90.0, 95.0, 99.0, 100.0}
	return &c
}

func (self *ColumnStat) add(cell string) {
	self.DistinctCntr[cell]++
	self.TotalCnt++
}

func (self *ColumnStat) calcPerc() map[string]float64 {
	rv := make(map[string]float64)
	for k, v := range self.DistinctCntr {
		rv[k] = (float64(v) / float64(self.TotalCnt)) * 100
	}
	self.DistinctPerc = rv
	return rv
}

func (self *ColumnStat) createTable() table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Value", "Count", "Percent"})
	self.calcPerc()
	for colName, colCnt := range self.DistinctCntr {
		t.AppendRow([]interface{}{colName, colCnt, fmt.Sprintf("%.2f", self.DistinctPerc[colName])})
	}
	t.SortBy([]table.SortBy{
		{Name: "Count", Mode: table.Dsc},
	})
	return t
}
func (self *ColumnStat) printTable() {
	w := self.createTable()
	fmt.Println("\n", self.tblName)
	w.Render()
}

type ColumnMesurable struct {
	ColName     string
	vals        []float64
	Percentiles []float64
	Quantiles   map[string]float64
}

func NewColumnMesurable(ColName string) *ColumnMesurable {
	var c ColumnMesurable
	c.ColName = ColName
	c.Percentiles = defaultPercentiles
	//c.Percentiles = []float64{50.0, 75.0, 85.0, 90.0, 95.0, 99.0, 100.0}
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
	tblName     string
	Percentiles []float64
	percentiles map[string]float64
}

func NewTableMesurable(tblName string) *TableMesurable {
	var c TableMesurable
	c.tblName = tblName
	c.Percentiles = defaultPercentiles
	//c.Percentiles = []float64{50.0, 75.0, 85.0, 90.0, 95.0, 99.0, 100.0}
	return &c
}

//func (self *ColumnsMesurable) calcPercentiles(arr []float64) map[string]float64 {
//	rv := make(map[string]float64)
//	for _, p := range self.Percentiles {
//		q, _ := stats.Percentile(arr, p)
//		rv[fmt.Sprintf("%.0f", p)] = q
//	}
//	self.percentiles = rv
//	return rv
//}

func (self *TableMesurable) createTable(metrics []*ColumnMesurable) table.Writer {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	header := table.Row{"Name"}
	for _, pVal := range self.Percentiles {
		pName := fmt.Sprintf("%.0f", pVal)
		header = append(header, pName+"%p")
	}
	t.AppendHeader(header)
	for _, m := range metrics {
		row := table.Row{m.ColName}
		m.calcPercentiles()
		for _, qVal := range m.Quantiles {
			row = append(row, fmt.Sprintf("%.0f", qVal))
		}
		t.AppendRow(row)
	}
	return t
}
func (self *TableMesurable) printTable(metrics []*ColumnMesurable) {
	w := self.createTable(metrics)
	fmt.Println("\n", self.tblName)
	w.Render()
}

func ProcTestResults(results []*client.Result) {
	colSucc := NewColumnStat("Success Rate")
	colErr := NewColumnStat("Errors")
	colTgtStatus := NewColumnStat("Taget HTTP status codes")
	colPrxStatus := NewColumnStat("Proxy HTTP status codes")

	tblLatency := NewTableMesurable("Latency")
	latTTFB := NewColumnMesurable("TTFB")
	latDNS := NewColumnMesurable("DNS resolve")
	//latTTFB := []float64{}
	//latDNS := []float64{}
	for _, r := range results {
		if r.Status {
			colSucc.add("ok")
		} else {
			colSucc.add("error")
		}
		errNorm, _ := r.Error.MarshalCSV()
		if errNorm == "" {
			colErr.add("ok")
		} else {
			colErr.add(errNorm)
		}
		colTgtStatus.add(strconv.Itoa(r.TargetStatusCode))
		colPrxStatus.add(strconv.Itoa(r.ProxyStatusCode))
		latTTFB.vals = append(latTTFB.vals, float64(r.Latency.TTFB))
		latDNS.vals = append(latDNS.vals, float64(r.Latency.DNSresolve))

	}
	colSucc.printTable()
	colErr.printTable()
	colTgtStatus.printTable()
	colPrxStatus.printTable()

	measurableMetrics := []*ColumnMesurable{latDNS, latTTFB}
	tblLatency.printTable(measurableMetrics)
}

//func main() {
//	resultsFile, err := os.Open("1k-soax-session.csv")
//	if err != nil {
//		panic(err)
//	}
//	defer resultsFile.Close()
//
//	reader := csv.NewReader(resultsFile)
//	rawData, err := reader.ReadAll()
//	if err != nil {
//		panic(err)
//	}
//	colSucc := NewColumnStat("Success Rate")
//	colErr := NewColumnStat("Errors")
//	colTgtStatus := NewColumnStat("Taget HTTP status codes")
//	colPrxStatus := NewColumnStat("Proxy HTTP status codes")
//	latTTFB := []float64{}
//	latDNS := []float64{}
//	for _, row := range rawData {
//		fmt.Println(row)
//		fmt.Println(row[2])
//		row11cell := "ok"
//		if row[11] != "" {
//			row11cell = row[11]
//		}
//		colSucc.add(row[1])
//		colErr.add(row11cell)
//		colTgtStatus.add(row[2])
//		colPrxStatus.add(row[3])
//		if row[4] != "" {
//			ttfbFloat64, _ := strconv.ParseFloat(row[4], 64)
//			latTTFB = append(latTTFB, ttfbFloat64)
//		}
//		if row[5] != "" {
//			DnsFloat64, _ := strconv.ParseFloat(row[5], 64)
//			latDNS = append(latDNS, DnsFloat64)
//		}
//	}
//	fmt.Println("Success", colSucc.DistinctCntr, colSucc.calcPerc())
//	fmt.Println("Errors", colErr.DistinctCntr, colErr.calcPerc())
//	fmt.Println("TgtCode", colTgtStatus.DistinctCntr, colTgtStatus.calcPerc())
//	fmt.Println("PrxCode", colPrxStatus.DistinctCntr, colPrxStatus.calcPerc())
//	//ttfbMedian, _ := stats.Median(latTTFB)
//	//fmt.Println("TTFB", ttfbMedian)
//	colTTFB := NewColumnMesurable("TTFB")
//	coDNS := NewColumnMesurable("TTFB")
//	colSucc.printTable()
//	colErr.printTable()
//	colTgtStatus.printTable()
//	colPrxStatus.printTable()
//
//	fmt.Println("TTFB", colTTFB.calcPercentiles(latTTFB))
//	fmt.Println("DNS", coDNS.calcPercentiles(latDNS))
//	colTTFB.printTable()
//}
