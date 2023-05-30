package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"ms-astrid/products/models"
	"ms-astrid/products/models/request"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const astridTotalPerPage = 1000
const astridClient = "astrid"
const algoliaTotalPerPage = 1000
const algoliaClient = "algolia"
const m2AlgoliaParameters = "m2"

type MergePriceService struct {
	BaseService
	clientsHttpConfig map[string]request.ClientHttpConfig
	algoliaParameters map[string]request.AlgoliaParameters
}

// booleanToInt
func booleanToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// getEndpoint Get an endpoint of required client
func (service *MergePriceService) getEndpoint(client string, endpoint string) (string, error) {
	fullEndpoint := ""
	var err error
	if clientConfig, clientExist := service.clientsHttpConfig[client]; clientExist {
		clientEndpoints := clientConfig.Endpoints
		if clientEndpoint, endpointExist := clientEndpoints[endpoint]; endpointExist {
			fullEndpoint = fmt.Sprintf("%s%s", clientConfig.BaseUrl, clientEndpoint)
		} else {
			err = errors.New("endpoint not found")
		}
	} else {
		err = errors.New("client not found")
	}

	return fullEndpoint, err
}

// getItemsAstridByPage Get item list of astrid by page
func (service *MergePriceService) getItemsAstridByPage(typeItem string, page int, limit int) request.ResponseAstridPriceList {
	var response request.ResponseAstridPriceList
	endpoint, _ := service.getEndpoint(astridClient, typeItem)
	paginate := fmt.Sprintf("limit=%d&page=%d", limit, page)
	endpointPaginate := fmt.Sprintf("%s?%s", endpoint, paginate)

	// Get response and close response on final
	resp, errResponse := http.Get(endpointPaginate)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}(resp.Body)

	if errResponse == nil {
		bodyBytes, errBody := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			log.Println(string(bodyBytes))
		}
		if errBody == nil {
			if errUnmarshal := json.Unmarshal(bodyBytes, &response); errUnmarshal != nil {
				log.Println(errBody.Error())
			}
		} else {
			log.Println(errBody.Error())
		}
	} else {
		log.Println(errResponse.Error())
	}

	return response
}

// getItemsAstrid Get item list of astrid
func (service *MergePriceService) getItemsAstrid(typeItem string, itemsBySku map[string]request.ItemPriceAstrid) map[string]request.ItemPriceAstrid {
	var items []request.ItemPriceAstrid
	page := 1
	response := service.getItemsAstridByPage(typeItem, page, astridTotalPerPage)
	items = append(items, response.Data...)
	totalPages := response.Pages

	var waitGroup sync.WaitGroup
	channel := make(chan request.ResponseAstridPriceList)
	page++

	for page <= totalPages {
		waitGroup.Add(1)
		if page%5 == 0 {
			sleepInt := 1
			sleepDurationString := os.Getenv("MERGE_SERVICE_SLEEP")
			if len(sleepDurationString) == 1 {
				sleepInt, _ = strconv.Atoi(sleepDurationString)
			}
			time.Sleep(time.Duration(sleepInt) * time.Second)
		}
		go func(typeItem string, noPage int, channel chan<- request.ResponseAstridPriceList, group *sync.WaitGroup) {
			defer group.Done()
			channel <- service.getItemsAstridByPage(typeItem, noPage, astridTotalPerPage)
		}(typeItem, page, channel, &waitGroup)
		page++
	}

	go func() {
		waitGroup.Wait()
		close(channel)
	}()

	for responseChannel := range channel {
		items = append(items, responseChannel.Data...)
	}

	for _, item := range items {
		if item.SkuParent != "" {
			item.Sku = item.SkuParent
			if item.Price <= 0 {
				continue
			}
		}
		itemsBySku[item.Sku] = item
	}

	return itemsBySku
}

// getItemsAlgoliaByPage Get item list of algolia by page
func (service *MergePriceService) getItemsAlgoliaByPage(algoliaConfigKey string, page int, limit int, cursor string) request.ResponseBrowseAlgolia {
	paginate := fmt.Sprintf("hitsPerPage=%d&page=%d", limit, page)
	endpointTemplate, _ := service.getEndpoint(algoliaClient, "search")
	parameters := service.algoliaParameters[algoliaConfigKey]
	endpoint := fmt.Sprintf(endpointTemplate, parameters.ApplicationId, parameters.Index, parameters.ApplicationId, parameters.ApiKey)
	endpointPaginate := fmt.Sprintf("%s&%s", endpoint, paginate)

	if cursor != "" {
		endpointPaginate = fmt.Sprintf("%s&cursor=%s", endpointPaginate, cursor)
	}

	// Get response and close response on final
	resp, errResponse := http.Get(endpointPaginate)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println(err.Error())
		}
	}(resp.Body)

	var response request.ResponseBrowseAlgolia
	if errResponse == nil {
		bodyBytes, errBody := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			log.Println(string(bodyBytes))
		}
		if errBody == nil {
			if errUnmarshal := json.Unmarshal(bodyBytes, &response); errUnmarshal != nil {
				log.Println(errBody.Error())
			}
		} else {
			log.Println(errBody.Error())
		}
	} else {
		log.Println(errResponse.Error())
	}

	return response

}

// getItemsAlgolia Get item list of algolia
func (service *MergePriceService) getItemsAlgolia(algoliaParameters string) map[string]request.ItemBrowseAlgolia {
	var items []request.ItemBrowseAlgolia
	page := 0
	response := service.getItemsAlgoliaByPage(algoliaParameters, page, algoliaTotalPerPage, "")
	items = append(items, response.Hits...)
	totalPages := response.Pages - 1

	var waitGroup sync.WaitGroup
	channel := make(chan request.ResponseBrowseAlgolia)
	page++

	for page <= (totalPages - 1) {
		waitGroup.Add(1)
		go func(algoliaParameters string, noPage int, channel chan<- request.ResponseBrowseAlgolia, group *sync.WaitGroup) {
			defer group.Done()
			channel <- service.getItemsAlgoliaByPage(algoliaParameters, noPage, astridTotalPerPage, "")
		}(algoliaParameters, page, channel, &waitGroup)
		page++
	}

	go func() {
		waitGroup.Wait()
		close(channel)
	}()

	var responseLastItems request.ResponseBrowseAlgolia
	for responseChannel := range channel {
		items = append(items, responseChannel.Hits...)
		if responseChannel.Page == (totalPages - 1) {
			responseLastItems = service.getItemsAlgoliaByPage(algoliaParameters, page, algoliaTotalPerPage, responseChannel.Cursor)
		}
	}

	if responseLastItems.Page != 0 {
		items = append(items, responseLastItems.Hits...)
	}

	var itemsBySku = make(map[string]request.ItemBrowseAlgolia)
	for _, item := range items {
		itemsBySku[item.Sku] = item
	}

	return itemsBySku
}

// getItemsAlgolia Get item list of algolia
func (service *MergePriceService) getItemsAlgoliaSec(algoliaParameters string) map[string]request.ItemBrowseAlgolia {
	var items []request.ItemBrowseAlgolia
	page := 0
	response := service.getItemsAlgoliaByPage(algoliaParameters, page, algoliaTotalPerPage, "")
	items = append(items, response.Hits...)
	totalPages := response.Pages - 1
	page++

	for page <= totalPages {
		response = service.getItemsAlgoliaByPage(algoliaParameters, page, astridTotalPerPage, response.Cursor)
		items = append(items, response.Hits...)
		page++
	}

	var itemsBySku = make(map[string]request.ItemBrowseAlgolia)
	for _, item := range items {
		itemsBySku[item.Sku] = item
	}

	return itemsBySku
}

// compareItems compare and merge items of algolia and astrid
func (service *MergePriceService) compareItems(
	astridItems map[string]request.ItemPriceAstrid,
	algoliaItems map[string]request.ItemBrowseAlgolia,
) []models.ItemMergePrice {
	var resultMerged []models.ItemMergePrice
	for sku, algoliaItem := range algoliaItems {
		if currentItem, itemExist := astridItems[sku]; itemExist {
			var newItem models.ItemMergePrice
			newItem.Sku = currentItem.Sku
			newItem.Name = algoliaItem.Name
			newItem.Price = currentItem.Price
			newItem.Msrp = currentItem.Msrp
			newItem.M2 = algoliaItem.Price.MXN.Default
			newItem.M2Change = booleanToInt(algoliaItem.Price.MXN.Default != currentItem.Price)
			newItem.Gama = "N/A"
			newItem.GamaChange = 0
			newItem.Cost = currentItem.Cost
			newItem.IsEnabled = algoliaItem.IsEnabled
			newItem.IsAvailable = algoliaItem.IsAvailable
			newItem.Comercializable = algoliaItem.EnhancedAttributes.Generales.Comercializable
			resultMerged = append(resultMerged, newItem)
		}
	}

	return resultMerged
}

// MergeSimpleAndGrouped items
func (service *MergePriceService) MergeSimpleAndGrouped(simpleSkus []models.Sku, groupedSkus []models.GaiaGroup) map[string]request.ItemPriceAstrid {
	var mappedSkus = make(map[string]request.ItemPriceAstrid)
	for _, simpleItem := range simpleSkus {
		var itemAstrid request.ItemPriceAstrid
		itemAstrid.Sku = simpleItem.Sku
		itemAstrid.Msrp = simpleItem.Msrp
		itemAstrid.Price = simpleItem.Price
		itemAstrid.Cost = simpleItem.Cost
		mappedSkus[simpleItem.Sku] = itemAstrid
	}

	for _, groupedItem := range groupedSkus {
		if groupedItem.Price > 0 {
			var itemAstrid request.ItemPriceAstrid
			itemAstrid.Sku = groupedItem.ParentSku
			itemAstrid.Msrp = groupedItem.Msrp
			itemAstrid.Price = groupedItem.Price
			itemAstrid.Cost = groupedItem.Cost
			mappedSkus[groupedItem.ParentSku] = itemAstrid
		}
	}

	return mappedSkus
}

// writeFileJson Write x results in json
func (service *MergePriceService) writeFileJson(name string, data []byte) (string, error) {
	routeFile := "/tmp/" + name + ".json"
	file, err := os.Create(routeFile)
	if err == nil {
		_, err = file.Write(data)
	}

	return routeFile, err
}

// segmentMergeInFiles all result of merge in files
func (service *MergePriceService) segmentMergeInFiles(mergedItems []models.ItemMergePrice) []string {
	var fileNames []string
	var batch [][]models.ItemMergePrice
	batchSize := 10000

	var to int
	for from := 0; from < len(mergedItems); from += batchSize {
		to += batchSize
		if to > len(mergedItems) {
			to = len(mergedItems)
		}
		batch = append(batch, mergedItems[from:to])
	}

	var waitGroup sync.WaitGroup
	channel := make(chan string)
	currentTime := strconv.Itoa(int(time.Now().Unix()))
	for idx, currentBatch := range batch {
		idxName := strconv.Itoa(idx + 1)
		fileName := currentTime + "_" + idxName
		currentBatchJson, _ := json.Marshal(currentBatch)
		waitGroup.Add(1)
		go func(channelWrite chan<- string, group *sync.WaitGroup) {
			defer group.Done()
			fileRoute, err := service.writeFileJson(fileName, currentBatchJson)
			if err != nil {
				log.Printf("Error trying to create json merged: %s", err.Error())
			} else {
				channelWrite <- fileRoute
			}
		}(channel, &waitGroup)
	}
	go func() {
		waitGroup.Wait()
		close(channel)
	}()

	for responseChannel := range channel {
		fileNames = append(fileNames, responseChannel)
	}

	return fileNames
}

// saveFile into bucket
func (service *MergePriceService) saveFile(urlPath string, fileName string) (map[string]interface{}, error) {
	var response map[string]interface{}
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileNameSplit := strings.Split(fileName, "/")
	fileJson := fileName[len(fileNameSplit)-1:]
	fw, err := writer.CreateFormFile("file", fileJson)
	if err == nil {
		file, err := os.Open(fileName)
		if err == nil {
			_, err := io.Copy(fw, file)
			if err == nil {
				err = writer.Close()
				if err == nil {
					req, err := http.NewRequest("POST", urlPath, bytes.NewReader(body.Bytes()))
					if err == nil {
						req.Header.Set("Content-Type", writer.FormDataContentType())
						responseHttp, _ := client.Do(req)
						if responseHttp.StatusCode != http.StatusOK {
							err = errors.New(fmt.Sprintf("Request failed with response code: %d", responseHttp.Body))
						} else {
							bodyBytes, err := io.ReadAll(responseHttp.Body)
							if err == nil {
								err = json.Unmarshal(bodyBytes, &response)
							}
						}
					}
				}
			}
		}
	}

	return response, err
}

// saveFiles save all json
func (service *MergePriceService) saveFiles(fileNames []string) []string {
	var links []string
	var waitGroup sync.WaitGroup
	channel := make(chan string)

	// S3
	pathSave := os.Getenv("PATH_SAVE")
	for _, name := range fileNames {
		waitGroup.Add(1)
		go func(channelWrite chan<- string, group *sync.WaitGroup, nameFile string) {
			defer group.Done()
			result, errSave := service.saveFile(pathSave, nameFile)
			if errSave != nil {
				log.Println(errSave.Error())
			}
			filesMap := result["files"].(map[string]interface{})
			fileMap := filesMap["file"].(map[string]interface{})
			channelWrite <- fileMap["link"].(string)
		}(channel, &waitGroup, name)
	}

	go func() {
		waitGroup.Wait()
		close(channel)
	}()

	for responseChannel := range channel {
		links = append(links, responseChannel)
	}

	return links
}

// GetPriceList Get merged price list of algolia and astrid
func (service *MergePriceService) GetPriceList(simpleSkus []models.Sku, groupedSkus []models.GaiaGroup) map[string]interface{} {
	// 1. Merge items astrid
	itemsAstrid := service.MergeSimpleAndGrouped(simpleSkus, groupedSkus)

	// 2. Get items algolia
	itemsAlgolia := service.getItemsAlgoliaSec(m2AlgoliaParameters)

	// 3. Merge items and set for paginator
	mergedItems := service.compareItems(itemsAstrid, itemsAlgolia)

	// 4. Segment in files
	fileNames := service.segmentMergeInFiles(mergedItems)

	// 5. Save and get links of files
	links := service.saveFiles(fileNames)

	return map[string]interface{}{
		"itemsProcessed": len(mergedItems),
		"files":          links,
	}
}

func NewMergePriceService() *MergePriceService {
	return &MergePriceService{
		clientsHttpConfig: map[string]request.ClientHttpConfig{
			astridClient: {
				BaseUrl: os.Getenv("ASTRID_BASE_URL"),
				Endpoints: map[string]string{
					"priceList":  "price-list",
					"gaiaGroups": "gaia-groups",
				},
			},
			algoliaClient: {
				BaseUrl: os.Getenv("ALGOLIA_BASE_URL"),
				Endpoints: map[string]string{
					"search": "1/indexes/%s/browse?X-Algolia-Application-Id=%s&X-Algolia-API-Key=%s",
				},
			},
		},
		algoliaParameters: map[string]request.AlgoliaParameters{
			m2AlgoliaParameters: {
				Index:         os.Getenv("ALGOLIA_INDEX"),
				ApplicationId: os.Getenv("ALGOLIA_APP_ID"),
				ApiKey:        os.Getenv("ALGOLIA_API_KEY"),
			},
		},
	}
}
