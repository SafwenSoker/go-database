package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"github.com/jcelliott/lumber"
)


const Version = "1.0.1"

type (
	Logger interface{
		Fatal(string, ...interface{})
		Error(string, ...interface{})
		Warn(string, ...interface{})
		Info(string, ...interface{})
		Debug(string, ...interface{})
		Trace(string, ...interface{})
	}
	Driver struct {
		mutex sync.Mutex 
		mutexes map[string]*sync.Mutex
		dir string
		log Logger
	}
)

type Options struct {
	Logger
}

func New(dir string, options *Options)(*Driver, error){
	dir = filepath.Clean(dir)

	opts := Options{}

	if options != nil{
		opts = *options
	}

	if opts.Logger == nil {
		opts.Logger = lumber.NewConsoleLogger((lumber.INFO))
	}

	driver := Driver{
		dir: dir,
		mutexes: make(map[string]*sync.Mutex),
		log: opts.Logger,
	}

	if _, err := os.Stat(dir); err != nil {
		opts.Logger.Debug("Using '%' (database already exists)\n", dir)
		return &driver, nil
	}

	opts.Logger.Debug("Creating the database at '%s'...\n",dir)
	return &driver,	os.MkdirAll(dir, 0755)
}

func (d *Driver) Write(collection,resource string, v interface{}) error{
	if collection == ""{
		return fmt.Errorf("Missing collection - no place to save the record")
	}

	if resource == "" {
		return fmt.Errorf("Missing resource - unable to save record (no name)!")
	}

	mutex := d.getOrCreateMutex(collection)

	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(d.dir,collection)
	finalPath := filepath.Join(dir, resource+".json")
	tmpPath := finalPath + ".tmp"

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(v, "","\t")
	if err != nil {
		return err
	}

	b = append(b,byte('\n'))

	if err := ioutil.WriteFile(tmpPath, b, 0644); err != nil {
		return err
	}

	return os.Rename(tmpPath, finalPath)
}

func (d *Driver) Read(collection, resource string, v interface{}) error {
	if collection == ""{
		return fmt.Errorf("Missing collection - unable to read!")
	}
	
	if resource == "" {
		return fmt.Errorf("Missing resource - unable to read record (no name)!")
	}

	record := filepath.Join(d.dir,collection,resource+".json")

	if _, err := stat(record); err != nil {
		return err
	}

	b, err := ioutil.ReadFile(record+".json")
	if err != nil{
		return err
	}

	return json.Unmarshal(b,&v)
}

func (d *Driver) ReadAll(collection string) ([]string, error) {
	if collection == ""{
		return nil, fmt.Errorf("Missing collection - unable to read!")
	}

	dir := filepath.Join(d.dir, collection)

	if _,err := stat(dir); err!= nil {
		return nil, err
	}

	
	files, _ := ioutil.ReadDir(dir)

	var records [] string

	for _,file := range files {
		b, err:= ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		records = append(records, string(b))
	}
	return records, nil


}


func (d *Driver) Delete(collection , resource string) error {
	path := filepath.Join(collection,resource)
	mutex := d.getOrCreateMutex(collection)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(d.dir, path)
	switch fi, err:= stat(dir); {
	case fi == nil,err != nil:
		return fmt.Errorf("Unable to find or directory named %v\n",path)
	case fi.Mode().IsDir():
		return os.RemoveAll(dir)
	case fi.Mode().IsRegular():
		return os.RemoveAll(dir+".json")
	}
	return nil 
	

}

func (d *Driver) getOrCreateMutex(collection string) *sync.Mutex{
	
	d.mutex.Lock()
	defer d.mutex.Unlock()
	
	m, ok := d.mutexes[collection]
	
	if !ok {
		m = &sync.Mutex{}
	}
		d.mutexes[collection] = m

	return m

}

func stat(path string)(fi os.FileInfo, err error){
	if fi, err = os.Stat(path); os.IsNotExist(err){
		fi, err = os.Stat(path + ".json")
	}
	return
}
type Address struct {
	City string
	State string
	Country string
	Pincode json.Number
}

type User struct {
	Name string
	Age json.Number
	Contact string
	Company string
	Address Address
}

func main(){
	dir := "./"

	db, err := New(dir, nil)

	if err != nil {
		fmt.Println("Error", err)
	}

	employees := []User{
		{"Safwen","23","safwen.soker@pyxis.com.tn", "Pyxis IT", Address{"Bardo","Tunis", "Tunisia","2000"}},
		{"Amine","23","amine.yahya@pyxis.com.tn", "Pyxis IT", Address{"Miami","Florida", "USA","2000"}},
		{"Ghayth","23","ghayth.zairi@pyxis.com.tn", "Pyxis IT", Address{"grombalia","Tunis", "Tunisia","2000"}},
		{"Houssem","23","houssem.korbi@pyxis.com.tn", "Pyxis IT", Address{"hawaria","Nabeul", "Tunisia","2000"}},
		{"Yassine","23","yassine.bensaid@pyxis.com.tn", "Pyxis IT", Address{"kelibia","Nabeul", "Tunisia","2000"}},
		{"Ahmed","23","ahmed.soltani@pyxis.com.tn", "Pyxis IT", Address{"zahra","Tunis", "Tunisia","2000"}},
	}

	for _, value := range employees{
		db.Write("users", value.Name, User{
			Name: value.Name,
			Age: value.Age,
			Contact: value.Contact,
			Company: value.Company,
			Address: value.Address,
		})
	}

	records, err := db.ReadAll("users")
	if err != nil {
		fmt.Println("Error", err)
	}

	fmt.Println(records)
	
	allusers := []User{}
	for _, record := range records {
		employeeFound := User{}
		if err := json.Unmarshal([]byte(record), &employeeFound); err != nil {
			fmt.Println("Error", err)
		}
		allusers = append(allusers,employeeFound)
	}

	fmt.Println((allusers))

	if err := db.Delete("users","Safwen"); err != nil {
		fmt.Println("Error", err)
	}

	if err := db.Delete("users",""); err != nil {
		fmt.Println("Error", err)
	}

}