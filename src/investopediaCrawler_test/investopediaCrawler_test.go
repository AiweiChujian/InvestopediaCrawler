package investopediaCrawler_test

import (
	"fmt"
	"main/src/investopediaCrawler"
	"net/http"
	"strconv"
	"testing"
)


func TestInvestopediaCrawler(t *testing.T) {
	results, err := investopediaCrawler.Fetch()
	if err != nil {
		fmt.Println(err)
		return
	}

	var data [] *investopediaCrawler.InvestopediaDoc
	for _, doc := range results{
		if doc != nil {
			data = append(data, doc)
			fmt.Printf("(%d)[%s]-[%d]\n", doc.DocId,doc.Author,doc.Updated)
		}
	}

	fmt.Printf("抓取完成, 需要抓取%d条, 成功抓取%d条\n", len(results), len(data))
	port := `:8080`
	mux := http.NewServeMux()
	for _, doc := range data {
		pattern := "/" + strconv.Itoa(doc.DocId)
		content := doc.Content
		mux.HandleFunc(pattern, func(writer http.ResponseWriter, req *http.Request) {
			writer.Write([]byte(content))
		})
		fmt.Println("=======================================")
		fmt.Println("标题:", doc.Title)
		fmt.Println("详情页:",doc.DetailLink)
		localPath := "http://127.0.0.1" + port + pattern
		fmt.Println("抓取正文:", localPath)
	}

	http.ListenAndServe(port, mux)
}

