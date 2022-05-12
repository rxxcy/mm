package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"github.com/PuerkitoBio/goquery"
)

var API_BASE_URL string
var wg sync.WaitGroup

// GetCurrentDirectory 获取基础路径
func GetCurrentDirectory() string {
	//返回绝对路径 filepath.Dir(os.Args[0])去除最后一个元素的路径
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println(err)
	}
	//将\替换成/
	return strings.Replace(dir, "\\", "/", -1)
}

// 检查文件夹是否存在不在则创建
func createDir(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		err := os.MkdirAll(path, 0766)
		if err != nil {
			return false
		}
		return true
	}
	if os.IsNotExist(err) {
		err := os.MkdirAll(path, 0766)
		if err != nil {
			return false
		}
		return true
	}
	return true
}

// 获取最大页码
func getMaxPage(url string) (maxPage int) {
	maxPage = 0
	response, err := http.Get(url)
	if err != nil {
		log.Fatalln("net error: ", url)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Printf("status code error: %d %s \n", response.StatusCode, response.Status)
		return
	}
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return
	}

	var pages []string
	doc.Find(".pagination a").Each(func(i int, selection *goquery.Selection) {
		page := selection.Text()
		pages = append(pages, page)
	})
	length := len(pages)
	if length <= 0 {
		return
	}
	tempMaxPage, _ := strconv.Atoi(pages[length-2])
	maxPage = tempMaxPage
	return
}

// 获取每页有几条数据
func getOnePage(page int, baseUrl string, baseDir string) {
	log.Printf("正在获取第 %d 页内容 \n", page)
	url := baseUrl + "/page/" + strconv.Itoa(page)
	response, err := http.Get(url)
	if err != nil {
		log.Fatalln("net error: ", url)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Printf("status code error: %d %s \n", response.StatusCode, response.Status)
		return
	}
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return
	}

	var items [][]string
	doc.Find("#features .blog-listing").Each(func(i int, selection *goquery.Selection) {
		item := selection.Find(".blog-title")
		if item != nil {
			a := item.Find("a")
			title := a.Text()
			href, _ := a.Attr("href")
			temp := make([]string, 2)
			temp[0] = title
			temp[1] = href
			items = append(items, temp)
		}
	})
	length := len(items)
	if length <= 0 {
		return
	}
	fmt.Printf("本页 %d 条数据 \n", length)
	//getItemContent()
	for _, item := range items {
		getItemContent(item, baseDir)
	}
}

// 获取某个套的内容
func getItemContent(item []string, basePath string) {
	log.Printf(" <%s> \n", item[0])
	url := item[1]
	response, err := http.Get(url)
	if err != nil {
		log.Fatalln("net error: ", url)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Printf("status code error: %d %s \n", response.StatusCode, response.Status)
		return
	}
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return
	}

	var images []string
	doc.Find(".blog-details-text p .img-responsive").Each(func(i int, selection *goquery.Selection) {
		src, _ := selection.Attr("src")
		images = append(images, src)
	})
	length := len(images)
	if length <= 0 {
		log.Fatalf(" 『 %s 』 好像没找到资料\n", item[0])
		return
	}
	tempPath := basePath + item[0]
	createDir(tempPath)
	for _index, _item := range images {
		tempIndex := strconv.Itoa(_index)
		fileName := tempPath + "/" + tempIndex + ".jpg"
		fmt.Printf("( %d/%d ) thread -> %s \n", _index+1, length, fileName)
		_item := _item
		wg.Add(1)
		go func() {
			defer wg.Done()
			downloadImage(_item, fileName)
		}()
	}
	wg.Wait()
}

// 保存图片
func downloadImage(url string, path string) {
	response, err := http.Get(url)
	if err != nil {
		log.Fatal("net error: ", url)
		return
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}
	_ = ioutil.WriteFile(path, body, 0755)
}

// func init() {
// 	fmt.Println("初始中...")
// 	// 
// 	fmt.Println("成功 (: ")
// }

func main() {
	log.SetPrefix("[rxxcy] ")
	var baseDir string
	tempBaseDir := GetCurrentDirectory()
	fmt.Printf("输入存储路径 ( 默认 %s , 最后不带 /): ", tempBaseDir)
	//fmt.Println(baseDir)
	_, err := fmt.Scanf("%s", &baseDir)
	if err != nil {
		return
	}
	if baseDir == "" {
		baseDir = tempBaseDir + "/image"
		log.Println("使用默认路径")
	}
	baseDir = baseDir + "/"
	//fmt.Println(tempBaseDir)
	API_BASE_URL := "https://mm.tvv.tw"
	maxPage := getMaxPage(API_BASE_URL)
	log.Printf("共 %d 页\n", maxPage)
	for i := 1; i < maxPage; i++ {
		getOnePage(i, API_BASE_URL, baseDir)
	}
}
