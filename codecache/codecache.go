package codecache

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	common "gjlim2485/bandwidthawarecaching/common"
	"os"
	"sync"
)

type PaddingPoint struct {
	File          string `json: file`
	StartingPoint int    `json: startingPoint`
}

func Encoding(data_amount int, filenames ...string) error {
	if len(filenames) != data_amount {
		return errors.New("missing data items to encode")
	}

	var data [][]byte
	var max_size int = 0            //will be used to determine padding
	var paddingPoint []PaddingPoint //used to determine the padding point of each data compared to max_size

	//we first load all the data into a 2d array for fetching
	for _, name := range filenames {
		x, err := loadData(name)
		if err != nil {
			return err
		}
		paddingPoint = append(paddingPoint, PaddingPoint{name, len(x)})
		if len(x) > max_size {
			max_size = len(x)
		}
		data = append(data, x)
	}

	//padding data
	var err error
	for i := range data {
		data[i], err = padData(data[i], max_size)
		if err != nil {
			return errors.New("error padding data")
		}
	}

	//encode the data
	for i := 1; i < len(data); i++ {
		data[0] = xorData(data[0], data[i])
	}

	//create .bin file for encoded data
	var encode_name string
	for _, i := range filenames {
		encode_name = encode_name + "-" + i
	}
	encode_name = encode_name[1:] //get rid of - in front
	file, err := os.Create(common.DataDirectory + "/" + encode_name + ".bin")
	if err != nil {
		return errors.New("error encoding data")
	}
	defer file.Close()

	err = binary.Write(file, binary.LittleEndian, data[0])
	if err != nil {
		return errors.New("error writing data to binary")
	}

	//create json header file for decoding
	jsonData, err := json.MarshalIndent(paddingPoint, "", "    ")
	if err != nil {
		return errors.New("error creating json file")
	}
	header_file, err := os.Create(common.DataDirectory + "/" + encode_name + ".json")
	if err != nil {
		return errors.New("error creating header data")
	}
	defer header_file.Close()
	_, err = header_file.Write(jsonData)
	if err != nil {
		return errors.New("error writing json data")
	}

	return nil
}

func Decoding(codedfile string, headerfile string, targetfile string) error {
	var header []PaddingPoint
	jsonData, err := os.ReadFile(common.DataDirectory + "/" + headerfile)
	if err != nil {
		return errors.New("error opening header json")
	}
	err = json.Unmarshal(jsonData, &header)
	if err != nil {
		return errors.New("error unmarshalling json")
	}

	encodedData, err := os.ReadFile(common.DataDirectory + "/" + codedfile)
	if err != nil {
		return errors.New("error opening encoded file")
	}
	max_size := len(encodedData) //used for padding decode
	var paddingRemoval int

	for _, s := range header {
		if s.File != targetfile {
			x, err := loadData(s.File)
			if err != nil {
				return errors.New("error loading data for decoding")
			}
			x, err = padData(x, max_size)
			if err != nil {
				return errors.New("error padding data for decoding")
			}
			encodedData = xorData(encodedData, x)
		} else {
			paddingRemoval = s.StartingPoint
		}
	}
	encodedData = encodedData[:paddingRemoval]

	err = os.WriteFile(common.DataDirectory+"/"+targetfile, encodedData, 0644)
	if err != nil {
		return errors.New("could not reconstruct data")
	}
	return nil
}

func loadData(filename string) ([]byte, error) {
	data, err := os.ReadFile(common.DataDirectory + "/" + filename)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func padData(file []byte, final_size int) ([]byte, error) {
	if len(file) > final_size {
		return nil, errors.New("file is bigger than final size")
	}
	amount_to_pad := final_size - len(file)
	file = append(file, make([]byte, amount_to_pad)...)
	return file, nil
}

func xorData(data1 []byte, data2 []byte) []byte {
	for i := 0; i < len(data1); i++ {
		data1[i] ^= data2[i]
	}
	return data1
}

func MakeGroups(userData []common.UserData) {
	var wg sync.WaitGroup
	groups := make(map[string][]common.UserData)
	for _, s := range userData {
		request := s.RequestData
		groups[request] = append(groups[request], s)
	}
	wg.Add(len(groups))
	dataChannel := make(chan common.UserIntersection, len(groups))
	for requestFile, userdata := range groups {
		go FindIntersection(&wg, userdata, requestFile, dataChannel) //use go routine to do it in parallel
	}
	wg.Wait()
	close(dataChannel)
	var intersectionCollection []common.UserIntersection
	for result := range dataChannel {
		intersectionCollection = append(intersectionCollection, result)
	}

	if common.EnableCodeCache {
		//need to find intersections once more
		//no need for go routine
		common.UserRequestTicketResult[common.UserRequestTicket] = FindRequestIntersection(intersectionCollection)
	} else {
		//return only single request intersection
		common.UserRequestTicketResult[common.UserRequestTicket] = intersectionCollection
	}
}

func FindIntersection(wg *sync.WaitGroup, userSets []common.UserData, requestFile string, resultCh chan<- common.UserIntersection) {
	defer wg.Done()
	var users []string
	var sets = make(map[string]int)
	for _, value := range userSets { //count every occurence of each local cache
		users = append(users, value.UserIP)
		for _, localcache := range value.LocalCache {
			sets[localcache] += 1
		}
	}

	total_set := len(userSets)
	var intersection []string
	for index, value := range sets {
		if value == total_set {
			intersection = append(intersection, index)
		}
	}

	resultCh <- common.UserIntersection{Users: users, Intersection: intersection, RequestFile: []string{requestFile}}
}

func FindRequestIntersection(intersectionSets []common.UserIntersection) []common.UserIntersection {
	group := make(map[string]common.CodedIntersection)
	var returnUser []common.UserIntersection
	//TODO: get cache list currentEdgeCache[]
	currentEdgeCache := common.EdgeCache.GetCacheList()
	//O(n) time complexity
	for _, s := range intersectionSets {
		var insertGroup common.CodedIntersection
		if common.StringinSlice(s.RequestFile[0], currentEdgeCache) {
			insertGroup.Users = s.Users
			for i := range s.Intersection {
				insertGroup.Intersection[s.Intersection[i]] = true
			}
			insertGroup.CodedFile[s.RequestFile[0]] = true
			group[s.RequestFile[0]] = insertGroup
		} else {
			//not in cache, need to replace
			returnUser = append(returnUser, common.UserIntersection{Users: s.Users, Intersection: []string{"miss"}, RequestFile: []string{s.RequestFile[0]}})
		}

	}

	//m * n complexity here (checking intersections becomes O(n^3))
	for _, s := range intersectionSets {
		for _, g := range group {
			if g.Intersection[s.RequestFile[0]] && FindInclusionGroup2Set(g.CodedFile, s.Intersection) {
				//found a group and set that can be coded
				g.Users = append(g.Users, s.Users...)
				g.CodedFile[s.RequestFile[0]] = true
				g.Intersection = FindMappedIntersection(g.Intersection, s.Intersection)
				delete(group, s.RequestFile[0])
				break
			}
		}
	}
	//group is now the final result
	//intersection no longer matters, just users and requestfiles
	for _, s := range group {
		users := s.Users
		var codedFile []string
		for i, k := range s.CodedFile {
			if k {
				codedFile = append(codedFile, i)
			}
		}
		returnUser = append(returnUser, common.UserIntersection{Users: users, Intersection: []string{"hit"}, RequestFile: codedFile})
	}
	return returnUser
}

func FindInclusionGroup2Set(items map[string]bool, set []string) bool {
	checkTotal := len(items)
	checkCurrent := 0
	for _, s := range set {
		if items[s] {
			checkCurrent++
			if checkCurrent == checkTotal {
				return true
			}
		}
	}
	return false
}

func FindMappedIntersection(setA map[string]bool, setB []string) map[string]bool {
	intersection := make(map[string]bool) // Initialize the map
	for _, s := range setB {
		if setA[s] {
			intersection[s] = true
		}
	}
	return intersection
}
