package metrics

import (
	"fmt"
	"net/http"
)

// Handler 返回 Prometheus 格式的服务器指标。
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	Global.FormatPrometheus(w)
	fmt.Fprint(w, "\n")
}
