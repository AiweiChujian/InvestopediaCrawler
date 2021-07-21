package investopediaCrawler

import (
	"fmt"
)

// Fetch 抓取方法
func Fetch() ([] *InvestopediaDoc, error) {
	sourceUrl := "https://www.investopedia.com/blockchain-4689765"
	body, err := FetchLink(sourceUrl)
	if err != nil {
		return  nil, err
	}
	docList, err := parseListPage(body)
	if err != nil {
		return nil, err
	}
	numberOfList := len(docList)
	fmt.Println("列表数据抓取完成", numberOfList)
	docCh := make(chan *InvestopediaDoc, numberOfList)
	for _, doc := range docList{
		go func(d *InvestopediaDoc, ch chan *InvestopediaDoc) {
			detailErr := fetchDetailFor(d)
			if detailErr != nil {
				fmt.Println("详情数据抓取失败", d.DetailLink, detailErr)
				ch <- nil
			} else {
				ch <- d
			}
		}(doc, docCh)
	}
	var ret [] *InvestopediaDoc
	for i := 0; i < numberOfList; i++ {
		ret = append(ret, <-docCh)
	}
	return ret, nil
}


// 抓取详情页
func fetchDetailFor(doc *InvestopediaDoc)  error {
	html, err := FetchLink(doc.DetailLink)
	if err != nil {
		return err
	}
	err = doc.parseDocMetaWithDetail(html)
	if err != nil {
		return err
	}
	return nil
}



