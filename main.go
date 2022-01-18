package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/PuerkitoBio/goquery"
)

type ContentFormat struct {
	Origin string `json:"origin"`
	Format string `json:"format"`
}

type CitationInfo struct {
	Id  string        `json:"id"`
	Ama ContentFormat `json:"ama"`
	Apa ContentFormat `json:"apa"`
	Mla ContentFormat `json:"mla"`
	Nlm ContentFormat `json:"nlm"`
}

func transTitleIntoURL(title string) string {
	escapeUrl := url.QueryEscape(title)
	return "https://pubmed.ncbi.nlm.nih.gov/?term=" + escapeUrl
}

func getRedirectURL(domain string, location string) string {
	return domain + location
}

func findFirstTitleURL(title string, r io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	var first_href_location string
	doc.Find(".docsum-title").Each(func(i int, s *goquery.Selection) {
		href_location, ok := s.Attr("href")
		if ok {
			fmt.Printf("href_location: %s\n", href_location)
			if i == 0 {
				first_href_location = href_location
			}
		}
	})

	fmt.Printf("first_href_location: %s\n", first_href_location)

	if len(first_href_location) == 0 {
		return "", fmt.Errorf("empty href location")
	}

	return getRedirectURL("https://pubmed.ncbi.nlm.nih.gov", first_href_location), nil
}

func pubMedicineSearch(title string, abstract_file *os.File, citation_file *os.File) {
	c := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	reqURL := transTitleIntoURL(title)
	fmt.Printf("reqURL: %s\n", reqURL)
	req, err := http.NewRequest("GET", reqURL, nil)

	resp, err := c.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	// 最终指向文章的URL链接地址
	var redirectedUrl string

	// 根据resp的返回码判断是否发生了页面重定向
	if resp.StatusCode == 302 {
		redirectLocation := resp.Header.Get("location")
		redirectedUrl = getRedirectURL("https://pubmed.ncbi.nlm.nih.gov", redirectLocation)
	} else if resp.StatusCode == 200 {
		// 没有根据标题找到具体的文章，需要从搜索结果中查找
		redirectedUrl, err = findFirstTitleURL(title, resp.Body)
		if err != nil {
			fmt.Errorf("title get first title: %s url failed\n", redirectedUrl)
			return
		}
	}

	fmt.Printf("Redirect URL: %s\n", redirectedUrl)

	res, err := http.Get(redirectedUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find(".abstract-content").Each(func(i int, s *goquery.Selection) {
		content := s.Find("p").Text()
		if len(content) != 0 {
			fmt.Printf("abstract-content:%d, %s\n", i, content)
			abstract_file.WriteString("-------------------------------------------\n")
			abstract_file.WriteString(content)
			abstract_file.WriteString(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n")
		}
	})

	citationURL := redirectedUrl + "/citations/"
	fmt.Printf("citation URL: %s\n", citationURL)

	//get citation info
	citation_res, err := http.Get(citationURL)
	if err != nil {
		log.Fatal(err)
	}
	defer citation_res.Body.Close()

	var citation CitationInfo
	s, _ := ioutil.ReadAll(citation_res.Body)
	err = json.Unmarshal(s, &citation)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("citation: %s\n", citation.Nlm.Format)
	citation_file.WriteString("-------------------------------------------\n")
	citation_file.WriteString(citation.Nlm.Format + "\n")
	citation_file.WriteString(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n")
}

// func createExcel() error {
// 	f := excelize.NewFile()
// 	// 创建一个工作表
// 	index := f.NewSheet("Sheet1")

// 	streamWriter, err := f.NewStreamWriter("Sheet1")
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	if err := streamWriter.SetRow("A1", []interface{}{
// 		excelize.Cell{Value: "Data"}}); err != nil {
// 		fmt.Println(err)
// 	}

// 	// 设置单元格的值
// 	f.SetCellValue("Sheet2", "A2", "Hello world.")
// 	f.SetCellValue("Sheet1", "B2", 100)
// 	// 设置工作簿的默认工作表
// 	f.SetActiveSheet(index)
// 	// 根据指定路径保存文件
// 	if err := f.SaveAs("result.xlsx"); err != nil {
// 		fmt.Println(err)
// 	}
// }

func main() {
	filePath := "./input.txt"
	abstractPath := "./abstract.txt"
	citationPath := "./citation.txt"

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Cannot open text file: %s, err: [%v]", filePath, err)
		return
	}
	defer file.Close()

	abstract_f, err := os.Create(abstractPath)
	if err != nil {
		log.Printf("Cannot create abstract file: %s, err: [%v]", abstractPath, err)
		return
	}
	defer abstract_f.Close()

	citation_f, err := os.Create(citationPath)
	if err != nil {
		log.Printf("Cannot create citation file: %s, err: [%v]", citationPath, err)
		return
	}
	defer citation_f.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		//do_your_function(line)
		fmt.Printf("%s\n", line)
		pubMedicineSearch(line, abstract_f, citation_f)
	}

	// url := transTitleIntoURL("Recent Progress in the Synergistic Combination of Nanoparticle-Mediated Hyperthermia and Immunotherapy for Treatment of Cancer")
	// fmt.Printf("request URL: %s\n", url)
	//pubMedicineSearch("Colorectal peritoneal metastases: Optimal management review")
}

// func main() {
// 	redirectedUrl := "https://pubmed.ncbi.nlm.nih.gov/?term=Colorectal+peritoneal+metastases%3A+Optimal+management+review"
// 	res, err := http.Get(redirectedUrl)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer res.Body.Close()

// 	if res.StatusCode != 200 {
// 		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
// 	}

// 	findFirstTitleURL("Colorectal peritoneal metastases: Optimal management review", res.Body)
// }
