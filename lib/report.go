package abat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"
)

// Reporter使用其他函数的返回的slice结果，生成一个bytes类型的slice报告
type Reporter func([]Result) ([]byte, error)

// ReportText返回一个量化结果并保存到文本，出错的话返回错误信息
func ReportText(results []Result) ([]byte, error) {
	m := NewMetrics(results)
	out := &bytes.Buffer{}

	w := tabwriter.NewWriter(out, 0, 8, 2, '\t', tabwriter.StripEscape)
	fmt.Fprintf(w, "Requests\t[total]\t%d\n", m.Requests)
	fmt.Fprintf(w, "Duration\t[total]\t%s\n", m.Duration)
	fmt.Fprintf(w, "QPS\t[mean]\t%f\n", m.QPS)
	fmt.Fprintf(w, "Latencies\t[mean, 50, 95, 99, max]\t%s, %s, %s, %s, %s\n",
		m.Latencies.Mean, m.Latencies.P50, m.Latencies.P95, m.Latencies.P99, m.Latencies.Max)
	//fmt.Fprintf(w, "Bytes In\t[total, mean]\t%d, %.2f\n", m.BytesIn.Total, m.BytesIn.Mean)
	fmt.Fprintf(w, "Bytes Out\t[total, mean]\t%d, %.2f\n", m.BytesOut.Total, m.BytesOut.Mean)
	fmt.Fprintf(w, "Success\t[ratio]\t%.2f%%\n", m.Success*100)
	fmt.Fprintf(w, "Status Codes\t[code:count]\t")
	for code, count := range m.StatusCodes {
		fmt.Fprintf(w, "%s:%d  ", code, count)
	}
	fmt.Fprintln(w, "\nError Set:")
	for _, err := range m.Errors {
		fmt.Fprintln(w, err)
	}

	if err := w.Flush(); err != nil {
		return []byte{}, err
	}
	return out.Bytes(), nil
}

// ReportJSON writes a computed Metrics struct to as JSON
func ReportJSON(results []Result) ([]byte, error) {
	return json.Marshal(NewMetrics(results))
}

// ReportPlot builds up a self contained HTML page with an interactive plot
// of the latencies of the requests. Built with http://dygraphs.com/
func ReportPlot(results []Result) ([]byte, error) {
	series := &bytes.Buffer{}
	for i, point := 0, ""; i < len(results); i++ {
		point = "[" + strconv.FormatFloat(
			results[i].Timestamp.Sub(results[0].Timestamp).Seconds(), 'f', -1, 32) + ","

		if results[i].Errors == "" {
			point += "NaN," + strconv.FormatFloat(results[i].Latency.Seconds()*1000, 'f', -1, 32) + "],"
		} else {
			point += strconv.FormatFloat(results[i].Latency.Seconds()*1000, 'f', -1, 32) + ",NaN],"
		}

		series.WriteString(point)
	}
	// Remove trailing commas
	if series.Len() > 0 {
		series.Truncate(series.Len() - 1)
	}

	return []byte(fmt.Sprintf(plotsTemplate, dygraphJSLibSrc(), series)), nil
}

const plotsTemplate = `<!doctype>
<html>
<head>
  <title>Abat Plots</title>
</head>
<body>
  <div id="latencies" style="font-family: Courier; width: 100%%; height: 600px"></div>
  <a href="#" download="stressplot.png" onclick="this.href = document.getElementsByTagName('canvas')[0].toDataURL('image/png').replace(/^data:image\/[^;]/, 'data:application/octet-stream')">Download as PNG</a>
  <script>
	%s
  </script>
  <script>
  new Dygraph(
    document.getElementById("latencies"),
    [%s],
    {
      title: 'Abat Plot',
      labels: ['Seconds', 'ERR', 'OK'],
      ylabel: 'Latency (ms)',
      xlabel: 'Seconds elapsed',
      showRoller: true,
      colors: ['#FA7878', '#8AE234'],
      legend: 'always',
      logscale: true,
      strokeWidth: 1.3
    }
  );
  </script>
</body>
</html>`
