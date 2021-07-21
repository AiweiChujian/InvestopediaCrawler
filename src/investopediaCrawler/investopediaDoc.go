package investopediaCrawler

import (
	"errors"
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"strconv"
	"strings"
)

type InvestopediaDoc struct {
	/** 列表页抓取数据 **/

	// 文章id
	DocId int
	// 详情页
	DetailLink string
	// 封面图
	CoverImg string
	// 标题
	Title string

	/** 详情页抓取数据 **/
	// 作者
	Author string
	// 更新时间
	Updated int64
	// 正文
	Content string
}

func (doc *InvestopediaDoc)parseDefaultArticle(root *html.Node, article *html.Node) {
	if nameNode := htmlquery.FindOne(article, `./header/div[2]/div[1]/div/a/span`); nameNode != nil {
		doc.Author = trimNodeText(htmlquery.InnerText(nameNode))
	} else if nameNode := htmlquery.FindOne(article, `./header/div[2]/div[1]/div/span`); nameNode != nil {
		doc.Author = trimNodeText(htmlquery.InnerText(nameNode))
	}

	// 更新时间
	if nodes := htmlquery.Find(article,`./header/div[2]/li/div`); nodes != nil {
		for _, node := range nodes{
			if htmlquery.SelectAttr(node,`class`) == "comp displayed-date" {
				tmp, err := timestampWithDateText(htmlquery.InnerText(node))
				if err != nil {
					panic(ERR_CANNOT_PARSE_UPDATED)
				}
				doc.Updated = tmp
			}
		}
	} else if node := htmlquery.FindOne(root, `//*[@id="displayed-date_1-0"]`); node != nil {
		tmp, err := timestampWithDateText(htmlquery.InnerText(node))
		if err != nil {
			panic(ERR_CANNOT_PARSE_UPDATED)
		}
		doc.Updated = tmp
	}
}

// 一种特殊的详情页, 示例: https://www.investopedia.com/articles/personal-finance/091316/top-3-books-learn-about-blockchain.asp
func (doc *InvestopediaDoc) parseStyle1Article(root *html.Node, article *html.Node) {
	namenode := htmlquery.FindOne(article, `//*[@id="mntl-byline__link_1-0"]/span`)
	doc.Author = htmlquery.InnerText(namenode)
	dateNode := htmlquery.FindOne(root, `//*[@id="displayed-date_1-0"]`)
	tmp, err := timestampWithDateText(htmlquery.InnerText(dateNode))
	if err != nil {
		panic(ERR_CANNOT_PARSE_UPDATED)
	}
	doc.Updated = tmp

}

func (doc *InvestopediaDoc) parseDocMetaWithDetail(htmlString string) (err error)  {
	defer func() {
		// 将异常恢复成错误
		if tmp:= recover(); tmp != nil {
			switch value := tmp.(type) {
			case error:
				err = value
			case string:
				err = errors.New(value)
			default:
				err = errors.New("parse detail error")
			}
		}
	}()

	root, err := htmlquery.Parse(strings.NewReader(htmlString))
	if err != nil {
		return err
	}

	// 解析元数据
	if article := htmlquery.FindOne(root,`//*[@id="article_1-0"]`); article != nil {
		doc.parseDefaultArticle(root, article)
	} else if article := htmlquery.FindOne(root,`//*[@id="mntl-external-basic-sublayout_1-0"]`); article != nil  {
		doc.parseStyle1Article(root, article)
	} else {
		return ERR_DETAIL_NO_PARSEFUNC
	}


	// 隐藏不需要的视图

	if head := htmlquery.FindOne(root,`/html/head`); head != nil {
		node := &html.Node{Type: html.ElementNode, Data: `script`}
		node.Attr = []html.Attribute{{Key: `type`, Val: `text/javascript` }}

		hideFunc := `const classList = [ "header mntl-block", "banner mntl-block", "footer mntl-block", "left-rail mntl-block", "article-sources mntl-block", "performance-marketing mntl-block", "related-recirc-section mntl-block", "textnote-placeholder mntl-block", "scads-to-load right-rail__item", "article-meta mntl-block", "article-header", "breadcrumbs", "mntl-leaderboard-header" ]; const hideElements = (classList) => { const fmtClassList = classList.map((x) => x .trim() .split(" ") .reduce((sum, y) => sum + "." + y, "") ); const style = document.createElement("style"); style.type = "text/css"; style.rel = "stylesheet"; style.appendChild( document.createTextNode(fmtClassList.join() + "{display:none}") ); const head = document.getElementsByTagName("head")[0]; head.appendChild(style); }; hideElements(classList);r`
		node.AppendChild(&html.Node{Type: html.TextNode, Data: hideFunc})
		head.AppendChild(node)
	}

	// 替换动态资源
	content := htmlquery.OutputHTML(root,true)
	srcHost := "https://www.investopedia.com"
	if head := htmlquery.FindOne(root,`/html/head`); head != nil {
		links := htmlquery.Find(head, `./link`)
		for _, link := range links{
			if isExistAtt(link, `data-glb-css`) {
				globalCssHref := htmlquery.SelectAttr(link, `href`)
				css, err := fetchLink(srcHost + globalCssHref)
				if err != nil {
					return err
				}
				oldNode := htmlquery.OutputHTML(link, true)
				newNode := `<style type="text/css">` + css + `</style>`
				content = strings.Replace(content, oldNode, newNode, 1)
				break
			}
		}
		scripts := htmlquery.Find(head, `./script`)
		for _, script := range scripts {
			if htmlquery.SelectAttr(script, `data-glb-js`) == "top" {
				topScriptSrc :=  htmlquery.SelectAttr(script, `src`)
				top, err := fetchLink(srcHost + topScriptSrc)
				if err != nil {
					return err
				}
				oldNode := htmlquery.OutputHTML(script, true)
				newNode := `<script type="text/javascript" data-glb-js="top">` + top + `</script>`
				content = strings.Replace(content, oldNode, newNode, 1)
				break
			}
		}
	}
	if body := htmlquery.FindOne(root,`/html/body`); body != nil {
		scripts := htmlquery.Find(body, `./script`)
		for _, script := range scripts {
			if htmlquery.SelectAttr(script, `data-glb-js`) == "bottom" {
				bottomScriptSrc :=  htmlquery.SelectAttr(script, `src`)
				bottom, err := fetchLink(srcHost + bottomScriptSrc)
				if err != nil {
					return err
				}
				oldNode := htmlquery.OutputHTML(script, true)
				newNode := `<script type="text/javascript" data-glb-js="bottom">` + bottom + `</script>`
				content = strings.Replace(content, oldNode, newNode, 1)
				break
			}
		}
	}
	doc.Content = content
	return err
}

// 解析列表页
func parseListPage(html string)(ret [] *InvestopediaDoc, err error) {
	defer func() {
		// 将异常恢复成错误
		if tmp:= recover(); tmp != nil {
			switch value := tmp.(type) {
			case error:
				err = value
			case string:
				err = errors.New(value)
			default:
				err = errors.New("parse list error")
			}
		}
	}()

	rootNode, err := htmlquery.Parse(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// heroCard
	heroCard :=  htmlquery.FindOne(rootNode,`//*[@id="hero-card_1-0"]`)
	idString := htmlquery.SelectAttr(heroCard, `data-doc-id`)
	docId, _ :=  strconv.Atoi(strings.Replace(idString,",","",-1))
	detailLink :=  htmlquery.SelectAttr(heroCard, `href`)
	coverImg := htmlquery.SelectAttr(htmlquery.FindOne(heroCard, `./div[1]/img`), `src`)
	title := htmlquery.InnerText(htmlquery.FindOne(heroCard, `./div[2]/span/span`))
	ret = append(ret, &InvestopediaDoc{DocId: docId, DetailLink: detailLink, CoverImg: coverImg, Title: title})
	// cardList
	cardList := htmlquery.FindOne(rootNode,`//*[@id="card-list_1-0"]`)
	nodes := htmlquery.Find(cardList,`./li`)
	for _, node := range nodes {
		card := htmlquery.FindOne(node, `./a]`)
		idString2 := htmlquery.SelectAttr(card, `data-doc-id`)
		docId2, _ :=  strconv.Atoi(strings.Replace(idString2,",","",-1))
		detailLink2 :=  htmlquery.SelectAttr(card, `href`)
		coverImg2 := htmlquery.SelectAttr(htmlquery.FindOne(card, `./div[1]/img`), `data-src`)
		title2 := htmlquery.InnerText(htmlquery.FindOne(card, `./div[2]/span/span`))
		ret = append(ret, &InvestopediaDoc{DocId: docId2, DetailLink: detailLink2, CoverImg: coverImg2, Title: title2})
	}

	// card list 2
	cardList2 := htmlquery.FindOne(rootNode,`//*[@id="card-list_2-0"]`)
	nodes2 := htmlquery.Find(cardList2,`./li`)
	for _, node := range nodes2 {
		card := htmlquery.FindOne(node, `./a]`)
		idString3 := htmlquery.SelectAttr(card, `data-doc-id`)
		docId3, _ :=  strconv.Atoi(strings.Replace(idString3,",","",-1))
		detailLink3 :=  htmlquery.SelectAttr(card, `href`)
		coverImg3 := htmlquery.SelectAttr(htmlquery.FindOne(card, `./div[1]/img`), `data-src`)
		title3 := htmlquery.InnerText(htmlquery.FindOne(card, `./div[2]/span/span`))
		ret = append(ret, &InvestopediaDoc{DocId: docId3, DetailLink: detailLink3, CoverImg: coverImg3, Title: title3})
	}
	return ret, err
}