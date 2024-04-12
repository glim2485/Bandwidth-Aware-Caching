package codecache

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	common "gjlim2485/bandwidthawarecaching/common"
	"os"
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
