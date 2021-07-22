package investopediaCrawler

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"regexp"
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

	root, err := html.Parse(strings.NewReader(htmlString))
	if err != nil {
		return err
	}

	// 解析元数据
	dom := goquery.NewDocumentFromNode(root)

	meta := dom.Find(`div.article-meta`).First()
	dateText := meta.Find(`.displayed-date`).Text()
	tmp, err := timestampWithDateText(dateText)
	if err != nil {
		return  err
	}
	doc.Updated = tmp
	if name := trimNodeText(meta.Find(`.mntl-byline__name a.mntl-byline__link`).First().Find(`.link__wrapper`).First().Text()); name != "" {
		doc.Author = name
	} else {
		doc.Author = trimNodeText(meta.Find(`.mntl-byline__name span.mntl-byline__span`).First().Text())
	}

	// 删除不需要的视图
	hideClassList := []string{
		".banner.mntl-block", ".footer.mntl-block", ".left-rail.mntl-block",
		".article-sources.mntl-block", ".performance-marketing.mntl-block",
		".related-recirc-section.mntl-block", ".textnote-placeholder.mntl-block",
		".scads-to-load.right-rail__item", ".article-meta.mntl-block", ".article-header",
		".breadcrumbs", ".mntl-leaderboard-header",
	}
	for _, class := range hideClassList {
		dom.Find(class).Each(func(_ int, selection *goquery.Selection) {
			for _, node := range selection.Nodes{
				node.Parent.RemoveChild(node)
			}
		})
	}

	// 隐藏不需要的视图(header会影响目录的展开, 所以不能删除, 而是隐藏)
	if headNodes := dom.Find(`:root>head`).First().Nodes; len(headNodes) > 0 {
		head := headNodes[0]
		node := &html.Node{Type: html.ElementNode, Data: `style`}
		node.Attr = []html.Attribute{{Key: `type`, Val: `text/css` }}
		cssText := `.header.mntl-block{display:none}`
		node.AppendChild(&html.Node{Type: html.TextNode, Data: cssText})
		head.AppendChild(node)
	}

	// 替换动态资源
	srcHost := "https://www.investopedia.com"
	if headSelection := dom.Find(`:root>head`).First(); len(headSelection.Nodes) > 0 {

		// 替换动态css样式
		if cssLink := headSelection.Find(`head>link[data-glb-css][href]`).First(); len(cssLink.Nodes) > 0 {
			globalCssHref, _ := cssLink.Attr("href")
			css, err := FetchLink(srcHost + globalCssHref)
			if err != nil {
				return err
			}

			rp := regexp.MustCompile(`(?U)url\(/static/.*\.(?:woff|woff2|ttf|svg)\)`)
			urlItems := rp.FindAllString(css, -1)
			for _, item := range urlItems {
				absHead := "url("+ srcHost + "/static/"
				newItem := strings.Replace(item, "url(/static/", absHead, 1)
				css = strings.Replace(css, item, newItem, 1)
			}

			oldNode := cssLink.Nodes[0]
			newNode := &html.Node{Type: html.ElementNode, Data: `style`}
			newNode.Attr = []html.Attribute{{Key: `type`, Val: `text/css` }}
			newNode.AppendChild(&html.Node{Type: html.TextNode, Data: css})
			oldNode.Parent.InsertBefore(newNode, oldNode)
			oldNode.Parent.RemoveChild(oldNode)
		}

		// 替换top scripts
		if topScript := headSelection.Find(`head>script[data-glb-js=top][src]`).First(); len(topScript.Nodes) > 0 {
			topScriptSrc, _ :=  topScript.Attr(`src`)
			topJs, err := FetchLink(srcHost + topScriptSrc)
			if err != nil {
				return err
			}

			oldNode := topScript.Nodes[0]
			newNode := &html.Node{Type: html.ElementNode, Data: `script`}
			newNode.Attr = []html.Attribute{{Key: `type`, Val: `text/javascript` }, {Key: `data-glb-js`, Val: `top` }}
			newNode.AppendChild(&html.Node{Type: html.TextNode, Data: topJs})
			oldNode.Parent.InsertBefore(newNode, oldNode)
			oldNode.Parent.RemoveChild(oldNode)
		}
	}

	if bodySelection := dom.Find(`:root>body`).First(); len(bodySelection.Nodes) > 0 {

		bodySelection.Find(`script[data-glb-js=bottom][src]~script`).Each(func(_ int, selection *goquery.Selection) {
			scriptText := selection.Text()
			rp := regexp.MustCompile(`(?s)Mntl\.utilities\.scriptsOnLoad\(document\.querySelectorAll\('script\[data-glb-js="bottom"]'\), function\(\) {(.*)}\);`)
			if !rp.MatchString(scriptText) {
				return
			}
			scriptNode := selection.Nodes[0]
			scriptNode.RemoveChild(scriptNode.FirstChild)
			trimScript := rp.FindStringSubmatch(scriptText)[1]
			scriptNode.AppendChild(&html.Node{Type: html.TextNode, Data: trimScript})
		})

		// 替换bottom scripts
		if bottomScript := bodySelection.Find(`body>script[data-glb-js=bottom][src]`).First(); len(bottomScript.Nodes) > 0 {
			bottomScriptSrc, _ :=  bottomScript.Attr(`src`)
			bottomJs, err := FetchLink(srcHost + bottomScriptSrc)
			if err != nil {
				return err
			}
			oldNode := bottomScript.Nodes[0]
			newNode := &html.Node{Type: html.ElementNode, Data: `script`}
			newNode.Attr = []html.Attribute{{Key: `type`, Val: `text/javascript` }, {Key: `data-glb-js`, Val: `bottom` }}
			newNode.AppendChild(&html.Node{Type: html.TextNode, Data: bottomJs})
			oldNode.Parent.InsertBefore(newNode, oldNode)
			oldNode.Parent.RemoveChild(oldNode)
		}
	}
	
	// 获取正文内容
	var buf bytes.Buffer
	err = html.Render(&buf, root)
	if err != nil {
		return err
	}
	doc.Content = buf.String()
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

	dom, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// heroCard
	dom.Find(`a.hero-card[data-doc-id][href]`).Each(func(idx int, selection *goquery.Selection) {
		doc := new(InvestopediaDoc)
		idString, _ := selection.Attr(`data-doc-id`)
		doc.DocId, _ =  strconv.Atoi(strings.Replace(idString,",","",-1))
		doc.DetailLink,_ = selection.Attr(`href`)
		doc.CoverImg, _ = selection.Find(`img.card__img[src]`).First().Attr(`src`)
		doc.Title = selection.Find(`.card__title-text`).First().Text()
		ret = append(ret, doc)
	})

	// cardList
	dom.Find(`ul.card-list`).Each(func(_ int, list *goquery.Selection) {
		list.Find(`li.card-list__item`).Each(func(_ int, item *goquery.Selection) {
			doc := new(InvestopediaDoc)
			card := item.Find(`a.card[data-doc-id][href]`).First()
			idString, _ := card.Attr(`data-doc-id`)
			doc.DocId, _ =  strconv.Atoi(strings.Replace(idString,",","",-1))
			doc.DetailLink,_ = card.Attr(`href`)
			doc.CoverImg, _ = card.Find(`img.card__img[src]`).First().Attr(`src`)
			doc.Title = card.Find(`.card__title-text`).First().Text()
			ret = append(ret, doc)
		})
	})

	return ret, err
}