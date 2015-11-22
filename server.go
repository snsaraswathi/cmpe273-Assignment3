package main
/*********************************************************************************************************

cost, distance and duration calculation include the cost/distance/duration to travel from the final destination (in the user's input) to the start location (i.e) cost, distance and duration are all ROUND TRIPS. 

	 To remove calculation of cost from final destination to the start position: uncomment line - 301 
	 To remove calculation of distance from final destination to the start position: uncomment lines - 310, 319 
	 To remove calculation of duration from final destination to the start position: uncomment line - 329

**********************************************************************************************************/
import (
    "os/exec"
    "encoding/json"
    "io/ioutil"
    "net/http"
    "fmt"
    "strconv"
    "os"
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "github.com/gorilla/mux"
    "log"
    "bytes"
)

type UberResponse struct {
	Prices         []*PriceEstimate `json:"prices"`
}

type PriceEstimate struct {
	ProductId       string  `json:"product_id"`
	CurrencyCode    string  `json:"currency_code"`
	DisplayName     string  `json:"display_name"`
	Estimate        string  `json:"estimate"`
	LowEstimate     int     `json:"low_estimate"`
	HighEstimate    int     `json:"high_estimate"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Duration        int     `json:"duration"`
	Distance        float64 `json:"distance"`
}

type Parameters struct {
	Start_latitude string  `json:"start_latitude"`
	Start_longitude string `json:"start_longitude"`
	End_latitude string `json:"end_latitude"`
	End_longitude string `json:"end_longitude"`
	Product_id string `json:"product_id"`
}
type Server struct {
  dbsession *mgo.Session
  dbcoll *mgo.Collection
}

type Person struct {
        Id string       `bson:"id"`
        Name    string
        Address string
        City string
        State string
        Zip string
        Coordinate      Coordinate //`bson:"inline"`
}

type Coordinate struct {
        Lat float64
        Lng float64
} 

type StartDestination struct{
	Starting_from_location_id string `json:"starting_from_location_id"`
	Location_ids []string `json:"location_ids"`
}

type NextDestination struct{
	Id string	`json:"id"`
        Status string	`json:"status"`
        Starting_from_location_id string	`json:"starting_from_location_id"`
        Next_destination_location_id string	`json:"next_destination_location_id"`
        Best_route_location_ids []string	`json:"best_route_location_ids"`
        Total_uber_costs int 	`json:"total_uber_costs"`
        Total_uber_duration int `json:"total_uber_duration"`
        Total_distance float64	`json:"total_distance"`
        Uber_wait_time_eta int 	`json:"uber_wait_time_eta"`
}

type Route struct{
	Id string
	Status string
	Starting_from_location_id string
	Best_route_location_ids []string
	Total_uber_costs int
	Total_uber_duration int
	Total_distance float64
}

type Request struct {
        RequestID       string  `json:"request_id"`
        Status          string `json:"status"`
        ETA             int     `json:"eta"`
        SurgeMultiplier float64 `json:"surge_multiplier"`
}

type PutRequest struct{
	StartLat string
	StartLng string
	EndLat string
	EndLng string
	ProductId string	
}

        var arr [100][100]int
	var durationArr [100][100]int
	var distanceArr [100][100]float64
	var minCost int=99999
        var LatLngArr [10]Coordinate
        var totalCost int = 0
	var totalDuration int= 0
	var bestTour[100] int
	var totalDistance float64= 0.0	
	
func main() {
        router := mux.NewRouter()
        router.HandleFunc("/tripplanner/", handlePostReq).Methods("POST")
	router.HandleFunc("/tripplanner/{id}", handleGetReq).Methods("GET")
	router.HandleFunc("/tripplanner/{id}", handlePutReq).Methods("PUT")
	http.ListenAndServe(":8080", router)
}
	
func establishDbConn() *Server{
       // session, err := mgo.Dial("mongodb://saro:1234@ds043714.mongolab.com:43714/contactdb")
          session, err := mgo.Dial("mongodb://saro:1234@ds055584.mongolab.com:55584/routedb")
	if err != nil {
                panic(err)
        }   
        session.SetMode(mgo.Monotonic, true)    
        //c := session.DB("contactdb").C("contactCollection")
        c := session.DB("routedb").C("routedbcollec")
        return &Server{dbsession:session,dbcoll:c}
}

func (s *Server) Close() {
  s.dbsession.Close()
}

func uniqueIDGen() string {
        out, err := exec.Command("uuidgen").Output()
        if err != nil {
                log.Fatal(err)
        }  
        s := string(out[:36])
        return s
}

func handlePostReq(res http.ResponseWriter, req *http.Request) {
        conn := establishDbConn()
        defer conn.dbsession.Close()
        c := conn.dbcoll

        res.Header().Set("Content-Type", "application/json")
	travelEndPoints := new(StartDestination)
	decoder := json.NewDecoder(req.Body)
        error := decoder.Decode(&travelEndPoints)
        if error != nil {
                        fmt.Fprint(res, "Input format invalid. Input should be in JSON format only!!")
                        return
                } 
	totalCost = 0
        totalDuration = 0
        totalDistance = 0.0
	result := Person{}
	err := c.Find(bson.M{"id": travelEndPoints.Starting_from_location_id}).One(&result)
        if err != nil {
                fmt.Fprint(res, "The requested id does not correspond to any entries")
                return
        }
	
	resultEndPoints := Person{}
	endPointsArray := travelEndPoints.Location_ids
	lengthOFArr := len(endPointsArray)
	
	LatLngArr[0].Lat = result.Coordinate.Lat
	LatLngArr[0].Lng = result.Coordinate.Lng
	
	for k :=1; k<=lengthOFArr;k++{
		errEnd := c.Find(bson.M{"id": endPointsArray[k-1]}).One(&resultEndPoints)
		if errEnd != nil {
                                fmt.Fprint(res, "The requested End id does not correspond to any entries")
                                return
                        }   
		LatLngArr[k].Lat = resultEndPoints.Coordinate.Lat
		LatLngArr[k].Lng = resultEndPoints.Coordinate.Lng
	}

	for i := 0; i <= lengthOFArr; i++ {
		for j := 0; j <= lengthOFArr; j++ {
			if(i == j){
				continue
			} else {
                	arr[i][j] = callUberApiForCost(LatLngArr[i].Lat, LatLngArr[i].Lng, LatLngArr[j].Lat, LatLngArr[j].Lng)
			}
		}
	}

	for x := 0; x <= lengthOFArr; x++ {
                for y := 0; y <= lengthOFArr; y++ {
                        if(x == y){ 
                                continue
                        } else {
                        distanceArr[x][y] = callUberApiForDistance(LatLngArr[x].Lat, LatLngArr[x].Lng, LatLngArr[y].Lat, LatLngArr[y].Lng)
                        }   
                }   
        }   


 	for a := 0; a <= lengthOFArr; a++ {
                for b := 0; b <= lengthOFArr; b++ {
                        if(a == b){ 
                                continue
                        } else {
                        durationArr[a][b] = callUberApiForDuration(LatLngArr[a].Lat, LatLngArr[a].Lng, LatLngArr[b].Lat, LatLngArr[b].Lng)
                        }   
                }   
        }   
 
	
	var tourArr [100]int
	for l:=0; l<lengthOFArr-1;l++{
		tourArr[l] = l+1
	}	
	getMinCost(tourArr, 0, lengthOFArr)
	var bestRouteId = make([]string,lengthOFArr)
	 
	for index:=0;index<lengthOFArr;index++{
		bestRouteId[index] = travelEndPoints.Location_ids[bestTour[index]-1]
	}
	
	calculateDistance(bestTour, lengthOFArr)
	calculateDuration(bestTour, lengthOFArr)
	
	responseVar := Route{}
	responseVar.Status= "planning"
	responseVar.Starting_from_location_id = travelEndPoints.Starting_from_location_id
	//responseVar.Best_route_location_ids = travelEndPoints.Location_ids
	responseVar.Best_route_location_ids=bestRouteId
	responseVar.Total_uber_costs = minCost
	responseVar.Total_uber_duration = totalDuration
	responseVar.Total_distance = totalDistance
	responseVar.Id=uniqueIDGen()
	errInsert := c.Insert(responseVar)
                if errInsert != nil {
                        panic(errInsert)
                }
	outgoingJSON, error := json.Marshal(responseVar)
        if error != nil {
                log.Println(error.Error())
                http.Error(res, error.Error(), http.StatusInternalServerError)
                return
        }
		res.WriteHeader(http.StatusCreated)
                fmt.Fprint(res, string(outgoingJSON))
}

func getMinCost(tour [100]int, index int, length int){
        if(index == length){
                       currCost := calculateCost(tour, length)   
			if(currCost<minCost){
				minCost=currCost
				for i:= 0; i<length;i++{
					bestTour[i]=tour[i]+1	
				}
                	}  else if(currCost == minCost){
				minDur := calculateDuration(bestTour, length)
				currDur := calculateCurrDur(tour, length)
				if(currDur < minDur){
					for i:= 0; i<length;i++{
						bestTour[i]=tour[i]+1	
					}
				}
			} 
        } else {
                for j := index; j<length; j++{
                        swap(&tour[index], &tour[j])    
                        getMinCost(tour, index+1, length)
                        swap(&tour[index], &tour[j])
                }   
        }   
}

func swap(px, py *int) {
        tempx := *px 
        tempy := *py 
        *px = tempy
        *py = tempx
}

func calculateCost(tour [100]int, length int)int{
	cost := arr[0][tour[0]+1]
 	for i := 0; i<length-1;i++ {
		cost = cost+arr[tour[i]+1][tour[i+1]+1]
	}
	cost += arr[tour[length-1]+1][0]	
	return cost
}

func calculateDistance(bestTour [100]int, length int)float64{
        totalDistance = distanceArr[0][bestTour[0]]
        for i := 0; i<length-1;i++ {
                totalDistance = totalDistance+distanceArr[bestTour[i]][bestTour[i+1]]
        }   
        totalDistance += distanceArr[bestTour[length-1]][0]   
	return totalDistance
}

func calculateCurrDur(tour [100]int, length int)int{
        totalDuration = durationArr[0][tour[0]+1]
        for i := 0; i<length-1;i++ {
                totalDuration = totalDuration+durationArr[tour[i]+1][tour[i+1]+1]
        }   
        totalDuration += durationArr[tour[length-1]+1][0]   
	return totalDuration
}


func calculateDuration(bestTour [100]int, length int)int {
        totalDuration = durationArr[0][bestTour[0]]
        for i := 0; i<length-1;i++ {
                totalDuration = totalDuration+durationArr[bestTour[i]][bestTour[i+1]]
        }   
        totalDuration += durationArr[bestTour[length-1]][0] 
	return totalDuration  
}

func callUberApiForCost(startLat float64, startLng float64, endLat float64, endLng float64) int{
	var params Parameters
	params.Start_latitude=strconv.FormatFloat(startLat, 'f', 6, 64)
	params.Start_longitude= strconv.FormatFloat(startLng, 'f', 6, 64)
	params.End_latitude=strconv.FormatFloat(endLat, 'f', 6, 64)
	params.End_longitude=strconv.FormatFloat(endLng, 'f', 6, 64)
	params.Product_id="bEWydlTqFbJhECIL4fULkyEWBiH6WGPz"
	apiResponse, err := http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=" +params.Start_latitude+ "&start_longitude=" +params.Start_longitude+ "&end_latitude=" +params.End_latitude+ "&end_longitude=" +params.End_longitude+ "&server_token=p1Rm-PUFDv52ZSvev4Xa1PFJjxTziTZZk2bZB9MA")
	
	var temp UberResponse

	if err != nil {
        fmt.Printf("%s", err)
        os.Exit(1)
        } else {
        defer apiResponse.Body.Close()
        contents, err := ioutil.ReadAll(apiResponse.Body)
        if err != nil {
            fmt.Printf("%s", err)
            os.Exit(1)
        }
        err1 := json.Unmarshal(contents, &temp)
        if err1 != nil {
              fmt.Printf("%s", err1)
              os.Exit(1)
          }
	err2 := json.Unmarshal(contents, &temp)
 	if err2 != nil {
              fmt.Printf("%s", err2)
              os.Exit(1)
          }
        }
	return temp.Prices[0].HighEstimate
}

func callUberApiForDistance(startLat float64, startLng float64, endLat float64, endLng float64) float64{
        var params Parameters
        params.Start_latitude=strconv.FormatFloat(startLat, 'f', 6, 64)
        params.Start_longitude= strconv.FormatFloat(startLng, 'f', 6, 64)
        params.End_latitude=strconv.FormatFloat(endLat, 'f', 6, 64)
        params.End_longitude=strconv.FormatFloat(endLng, 'f', 6, 64)
        params.Product_id="bEWydlTqFbJhECIL4fULkyEWBiH6WGPz"
        apiResponse, err := http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=" +params.Start_latitude+ "&start_longitude=" +params.Start_longitude+ "&end_latitude=" +params.End_latitude+ "&end_longitude=" +params.End_longitude+ "&server_token=p1Rm-PUFDv52ZSvev4Xa1PFJjxTziTZZk2bZB9MA")
        var temp UberResponse
        if err != nil {
        fmt.Printf("%s", err)
        os.Exit(1)
        } else {
        defer apiResponse.Body.Close()
        contents, err := ioutil.ReadAll(apiResponse.Body)
        if err != nil {
            fmt.Printf("%s", err)
            os.Exit(1)
        }
        err1 := json.Unmarshal(contents, &temp)
        if err1 != nil {
              fmt.Printf("%s", err1)
              os.Exit(1)
          }
        err2 := json.Unmarshal(contents, &temp)
        if err2 != nil {
              fmt.Printf("%s", err2)
              os.Exit(1)
          }
        }
        return temp.Prices[0].Distance
}

func callUberApiForDuration(startLat float64, startLng float64, endLat float64, endLng float64) int{
        var params Parameters
        params.Start_latitude=strconv.FormatFloat(startLat, 'f', 6, 64)
        params.Start_longitude= strconv.FormatFloat(startLng, 'f', 6, 64)
        params.End_latitude=strconv.FormatFloat(endLat, 'f', 6, 64)
        params.End_longitude=strconv.FormatFloat(endLng, 'f', 6, 64)
        params.Product_id="bEWydlTqFbJhECIL4fULkyEWBiH6WGPz"
        apiResponse, err := http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=" +params.Start_latitude+ "&start_longitude=" +params.Start_longitude+ "&end_latitude=" +params.End_latitude+ "&end_longitude=" +params.End_longitude+ "&server_token=p1Rm-PUFDv52ZSvev4Xa1PFJjxTziTZZk2bZB9MA")
        var temp UberResponse
        if err != nil {
        fmt.Printf("%s", err)
        os.Exit(1)
        } else {
        defer apiResponse.Body.Close()
        contents, err := ioutil.ReadAll(apiResponse.Body)
        if err != nil {
            fmt.Printf("%s", err)
            os.Exit(1)
        }
        err1 := json.Unmarshal(contents, &temp)
        if err1 != nil {
              fmt.Printf("%s", err1)
              os.Exit(1)
          }
        err2 := json.Unmarshal(contents, &temp)
        if err2 != nil {
              fmt.Printf("%s", err2)
              os.Exit(1)
          }
        }
        return temp.Prices[0].Duration
}


func handleGetReq(res http.ResponseWriter, req *http.Request) {
        conn := establishDbConn()
        defer conn.dbsession.Close()
        c := conn.dbcoll

        res.Header().Set("Content-Type", "application/json")
        res.Header().Set("Access-Control-Allow-Origin", "*")
	result := Route{}
        vars := mux.Vars(req)

        err := c.Find(bson.M{"id": vars["id"]}).One(&result)
        if err != nil {
                fmt.Fprint(res, "The requested id does not correspond to any entries")
                return
        } 

        outgoingJSON, error := json.Marshal(result)
        if error != nil {
                log.Println(error.Error())
                http.Error(res, error.Error(), http.StatusInternalServerError)
                return
        } 
                fmt.Fprint(res, string(outgoingJSON))
}

func handlePutReq(res http.ResponseWriter, req *http.Request) {	
        conn := establishDbConn()
        defer conn.dbsession.Close()
        c := conn.dbcoll

        res.Header().Set("Content-Type", "application/json")
        res.Header().Set("Access-Control-Allow-Origin", "*")
        result := NextDestination{}
        vars := mux.Vars(req)

        err := c.Find(bson.M{"id": vars["id"]}).One(&result)
        if err != nil {
                fmt.Fprint(res, "The requested id does not correspond to any entries")
		return
        }
	
	if(result.Status == "complete"){
		fmt.Fprint(res, "Transaction complete.. You reached your start destination.. See you again later")
		return
	}
	startLatLng := Person{}
        errStartLatLng := c.Find(bson.M{"id": result.Starting_from_location_id}).One(&startLatLng)
        if errStartLatLng != nil {
                fmt.Fprint(res, "The requested id does not correspond to any entries")
                return
        }
	
	endLatLng := Person{}
	errEndLatLng := c.Find(bson.M{"id": result.Best_route_location_ids[0]}).One(&endLatLng)
	if errEndLatLng != nil {
		fmt.Fprint(res, "The requested id does not correspond to any entries")
		return
	}

	putReqVar :=PutRequest{}
        putReqVar.StartLat = strconv.FormatFloat(startLatLng.Coordinate.Lat, 'f', 6, 64)
        putReqVar.StartLng = strconv.FormatFloat(startLatLng.Coordinate.Lng, 'f', 6, 64)
 	putReqVar.EndLat = strconv.FormatFloat(endLatLng.Coordinate.Lat, 'f', 6, 64)
	putReqVar.EndLng = strconv.FormatFloat(endLatLng.Coordinate.Lng, 'f', 6, 64)
	

	 apiGetResp, errApiGet := http.Get("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=" + putReqVar.StartLat +"&start_longitude=" + putReqVar.StartLng + "&end_latitude="+ putReqVar.EndLat +"&end_longitude=" + putReqVar.EndLng + "&server_token=p1Rm-PUFDv52ZSvev4Xa1PFJjxTziTZZk2bZB9MA")
        if errApiGet != nil {
       	 	fmt.Printf("%s", errApiGet)
        os.Exit(1)
        } 
        defer apiGetResp.Body.Close()
        getContents, getErr := ioutil.ReadAll(apiGetResp.Body)
        if getErr != nil {
            fmt.Printf("%s", getErr)
            os.Exit(1)
        }
        var temp UberResponse
        err1 := json.Unmarshal(getContents, &temp)
        if err1 != nil {
              fmt.Printf("%s", err1)
              os.Exit(1)
          }
        putReqVar.ProductId = temp.Prices[0].ProductId

	productId := putReqVar.ProductId
	destLatitude := putReqVar.StartLat
	destLongitude := putReqVar.StartLng
	
	urlPost := "https://sandbox-api.uber.com/v1/requests?start_latitude=" + destLatitude + "&start_longitude=" + destLongitude + "&product_id=" + productId
    
	requestBody := []byte(`{"start_longitude":"` + destLongitude + `", "start_latitude":"` + destLatitude + `", "product_id":"` + productId + `"}`)
	apiPost, errPost := http.NewRequest("POST", urlPost, bytes.NewBuffer(requestBody))

        apiPost.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3RfcmVjZWlwdCIsInJlcXVlc3QiLCJoaXN0b3J5X2xpdGUiXSwic3ViIjoiZjExNzVhOTItY2UyNC00NTc1LThjZjctNjZlYjA5OWJmMDk2IiwiaXNzIjoidWJlci11czEiLCJqdGkiOiI1OWMyNzMwMy1jYTU0LTQ2OTAtOGM3MC01MWNiMjE0MTZhZTQiLCJleHAiOjE0NTA1NTI1ODgsImlhdCI6MTQ0Nzk2MDU4NywidWFjdCI6IkJDUjRicDJaMm9zd1dWaTd2SzFaWUR6WDlvR1pjcCIsIm5iZiI6MTQ0Nzk2MDQ5NywiYXVkIjoiYkVXeWRsVHFGYkpoRUNJTDRmVUxreUVXQmlINldHUHoifQ.mKLtDakuJscAo1G2i469hvhvdoBGNRLzfHZgIy1XzHqkgNBBjh4dYfWrffDO5g8QTFB1nDSpgTlHUSeCGP9-wzL4C1xwQV4lb5xQ4TJzIxYCVXR94uX0Wc-yE8bxuvEV7zjDRro4ccMZwUw9K1ZBC2_AWq2dxtLv6FPiEeN3xri06W4o2A_s7lUWQnRC7MNKQ7_MtYL8sKd_HnLOVMxrmngWvehUfHchQpcy1cCRhQgDhx8zOhmCowQZjvrk_YaJ9mVsuzUPPW-8eOlp3TWbcgGoZ4DMu7jsvJeHAKSXDTfujYKee6b6HVdQ8cSfROBOgYs3-HUzbDUI8Ix1DGNNUQ")

        apiPost.Header.Set("Content-Type", "application/json")

	if errPost != nil {
        	fmt.Printf("%s", errPost)
        	os.Exit(1)
        }      

        client := &http.Client{}
        resp1, errCli := client.Do(apiPost)
        if(errCli != nil) {
                panic(err)
        }
        defer resp1.Body.Close()
        bodyResp, _ :=ioutil.ReadAll(resp1.Body)

        var postResponse Request
        errPostResp := json.Unmarshal(bodyResp, &postResponse)
        if(errPostResp != nil) {
                panic(errPostResp)
        }      

	reqBodyForPut := []byte(`{"status":"accepted"}`)
        apiPut, errPut := http.NewRequest("PUT", "https://sandbox-api.uber.com/v1/sandbox/requests/"+postResponse.RequestID, bytes.NewBuffer(reqBodyForPut))
        apiPut.Header.Set("Authorization", "bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3RfcmVjZWlwdCIsInJlcXVlc3QiLCJoaXN0b3J5X2xpdGUiXSwic3ViIjoiZjExNzVhOTItY2UyNC00NTc1LThjZjctNjZlYjA5OWJmMDk2IiwiaXNzIjoidWJlci11czEiLCJqdGkiOiI1OWMyNzMwMy1jYTU0LTQ2OTAtOGM3MC01MWNiMjE0MTZhZTQiLCJleHAiOjE0NTA1NTI1ODgsImlhdCI6MTQ0Nzk2MDU4NywidWFjdCI6IkJDUjRicDJaMm9zd1dWaTd2SzFaWUR6WDlvR1pjcCIsIm5iZiI6MTQ0Nzk2MDQ5NywiYXVkIjoiYkVXeWRsVHFGYkpoRUNJTDRmVUxreUVXQmlINldHUHoifQ.mKLtDakuJscAo1G2i469hvhvdoBGNRLzfHZgIy1XzHqkgNBBjh4dYfWrffDO5g8QTFB1nDSpgTlHUSeCGP9-wzL4C1xwQV4lb5xQ4TJzIxYCVXR94uX0Wc-yE8bxuvEV7zjDRro4ccMZwUw9K1ZBC2_AWq2dxtLv6FPiEeN3xri06W4o2A_s7lUWQnRC7MNKQ7_MtYL8sKd_HnLOVMxrmngWvehUfHchQpcy1cCRhQgDhx8zOhmCowQZjvrk_YaJ9mVsuzUPPW-8eOlp3TWbcgGoZ4DMu7jsvJeHAKSXDTfujYKee6b6HVdQ8cSfROBOgYs3-HUzbDUI8Ix1DGNNUQ")
        apiPut.Header.Set("Content-Type", "application/json")
        if errPut != nil {
        	fmt.Printf("%s", errPut)
        os.Exit(1)
        }
        client1 := &http.Client{}
        resp2, errCliPut := client1.Do(apiPut)
        if(errCliPut != nil) {
                panic(errCliPut)
        }
        defer resp2.Body.Close()

	apiPostRep, errPostRep := http.NewRequest("POST", urlPost, bytes.NewBuffer(requestBody))
	apiPostRep.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3RfcmVjZWlwdCIsInJlcXVlc3QiLCJoaXN0b3J5X2xpdGUiXSwic3ViIjoiZjExNzVhOTItY2UyNC00NTc1LThjZjctNjZlYjA5OWJmMDk2IiwiaXNzIjoidWJlci11czEiLCJqdGkiOiI1OWMyNzMwMy1jYTU0LTQ2OTAtOGM3MC01MWNiMjE0MTZhZTQiLCJleHAiOjE0NTA1NTI1ODgsImlhdCI6MTQ0Nzk2MDU4NywidWFjdCI6IkJDUjRicDJaMm9zd1dWaTd2SzFaWUR6WDlvR1pjcCIsIm5iZiI6MTQ0Nzk2MDQ5NywiYXVkIjoiYkVXeWRsVHFGYkpoRUNJTDRmVUxreUVXQmlINldHUHoifQ.mKLtDakuJscAo1G2i469hvhvdoBGNRLzfHZgIy1XzHqkgNBBjh4dYfWrffDO5g8QTFB1nDSpgTlHUSeCGP9-wzL4C1xwQV4lb5xQ4TJzIxYCVXR94uX0Wc-yE8bxuvEV7zjDRro4ccMZwUw9K1ZBC2_AWq2dxtLv6FPiEeN3xri06W4o2A_s7lUWQnRC7MNKQ7_MtYL8sKd_HnLOVMxrmngWvehUfHchQpcy1cCRhQgDhx8zOhmCowQZjvrk_YaJ9mVsuzUPPW-8eOlp3TWbcgGoZ4DMu7jsvJeHAKSXDTfujYKee6b6HVdQ8cSfROBOgYs3-HUzbDUI8Ix1DGNNUQ")

        apiPostRep.Header.Set("Content-Type", "application/json")

if errPostRep != nil {
        fmt.Printf("%s", errPostRep)
        os.Exit(1)
        }

        clientRep := &http.Client{}
        resp1Rep, errCliRep := clientRep.Do(apiPostRep)
        if(errCliRep != nil) {
                panic(errCliRep)
        }
        defer resp1Rep.Body.Close()
        bodyRespRep, _ :=ioutil.ReadAll(resp1Rep.Body)

        var postResponseRep Request
        errPostRespRep := json.Unmarshal(bodyRespRep, &postResponseRep)
        if(errPostRespRep != nil) {
                panic(errPostRespRep)
        }

	response := NextDestination{}
	response.Id = result.Id
	response.Status = "requesting"
	response.Starting_from_location_id = result.Starting_from_location_id
	response.Best_route_location_ids = result.Best_route_location_ids
	response.Total_uber_costs = result.Total_uber_costs
	response.Total_uber_duration =result.Total_uber_duration
	response.Total_distance=result.Total_distance
	response.Uber_wait_time_eta = postResponseRep.ETA
	
	flag := true
	length := len(result.Best_route_location_ids)
	if(result.Next_destination_location_id == result.Best_route_location_ids[length-1]){
		flag = false
	} else if(result.Next_destination_location_id == ""){
		response.Next_destination_location_id = result.Best_route_location_ids[0]	
	}  else {
		tempNextLoc := result.Next_destination_location_id
		for i:=0; i<len(result.Best_route_location_ids); i++{
			if(tempNextLoc == result.Best_route_location_ids[i]){
				 response.Next_destination_location_id = result.Best_route_location_ids[i+1]
				 break
			} else {
				continue
			}
		}
	}
	if(flag == true){
	colQuerier := bson.M{"id": vars["id"]}
        change := bson.M{"$set": bson.M{"status":"requesting","next_destination_location_id": response.Next_destination_location_id, "uber_wait_time_eta": response.Uber_wait_time_eta}}
                UpdateErr := c.Update(colQuerier, change)
                if UpdateErr != nil {
                        fmt.Fprint(res, "The requested id does not correspond to any entries")
                        return
                }     
	} else if(flag == false){
		//response.Status = "requesting"
		response.Next_destination_location_id = result.Starting_from_location_id
		colQuerier := bson.M{"id": vars["id"]}
		change := bson.M{"$set": bson.M{"status":"complete","next_destination_location_id": result.Starting_from_location_id, "uber_wait_time_eta": response.Uber_wait_time_eta}}
		UpdateErr := c.Update(colQuerier, change)
		if UpdateErr != nil {
			fmt.Fprint(res, "The requested id does not correspond to any entries")
			return
		}
	}
	outgoingJSON, error := json.Marshal(response)
        if error != nil {
                log.Println(error.Error())
                http.Error(res, error.Error(), http.StatusInternalServerError)
                return
        }
                fmt.Fprint(res, string(outgoingJSON))
}
