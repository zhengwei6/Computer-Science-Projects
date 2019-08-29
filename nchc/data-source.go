package nchc

import (
	"io/ioutil"
)

// DataSource data source
type DataSource struct {
	Name string `json:"name"`
}

// NewDataSource new DataSource
func NewDataSource() *DataSource {
	return &DataSource{
		Name: "",
	}
}

// DataSourceList data srouce list
type DataSourceList struct {
	SourceList []*DataSource `json:"sourceList"`
}

// NewDataSourceList new DataSourceList
func NewDataSourceList() *DataSourceList {
	return &DataSourceList{
		SourceList: make([]*DataSource, 0),
	}
}

// ReadDataSourceFromDir read data source from dir
func ReadDataSourceFromDir(res *DataSourceList) error {
	files, err := ioutil.ReadDir(dataSourcePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		// fmt.Println(file.Name())
		if file.Mode().IsDir() {
			// check if the directory is valid
			s := NewDataSource()
			s.Name = file.Name()
			res.SourceList = append(res.SourceList, s)
		}
	}
	return nil
}
