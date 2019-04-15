// Copyright 2019 The xvalue-dccp 


package dccp

import(
	"strings"
	"errors"
	"runtime"
	"path/filepath"
	"os"
	"os/user"
	"github.com/xvalue/go-xvalue/ethdb"
	"github.com/xvalue/go-xvalue/crypto"
	"github.com/xvalue/go-xvalue/log"
)


func GetDbDir() string {
    if datadir != "" {
	return datadir+"/dccpdb"
    }

    ss := []string{"dir",cur_enode}
    dir = strings.Join(ss,"-")
    return dir
}

func DefaultDataDir() string {
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "XValue")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "XValue")
		} else {
			return filepath.Join(home, ".xvalue")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

//for lockout info 
func GetDbDirForLockoutInfo() string {

    if datadir != "" {
	return datadir+"/lockoutinfo"
    }

    s := DefaultDataDir()
    log.Debug("==========GetDbDirForLockoutInfo,","datadir",s,"","===========")
    s += "/lockoutinfo"
    return s
}

//for write dccpaddr 
func GetDbDirForWriteDccpAddr() string {

    if datadir != "" {
	return datadir+"/dccpaddrs"
    }

    s := DefaultDataDir()
    log.Debug("==========GetDbDirForWriteDccpAddr,","datadir",s,"","===========")
    s += "/dccpaddrs"
    return s
}

//for node info save
func GetDbDirForNodeInfoSave() string {

    if datadir != "" {
	return datadir+"/nodeinfo"
    }

    s := DefaultDataDir()
    log.Debug("==========GetDbDirForNodeInfoSave,","datadir",s,"","===========")
    s += "/nodeinfo"
    return s
}

//for lockin
func GetDbDirForLockin() string {
    if datadir != "" {
	return datadir+"/hashkeydb"
    }

    ss := []string{"dir",cur_enode}
    dir = strings.Join(ss,"-")
    dir += "-"
    dir += "hashkeydb"
    return dir
}

func SetDatadir (data string) {
	datadir = data
}

func GetLockoutInfoFromLocalDB(hashkey string) (string,error) {
    if hashkey == "" {
	return "",errors.New("param error get lockout info from local db by hashkey.")
    }
    
    lock5.Lock()
    path := GetDbDirForLockoutInfo()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============GetLockoutInfoFromLocalDB,create db fail.============")
	lock5.Unlock()
	return "",errors.New("create db fail.")
    }
    
    value,has:= db.Get([]byte(hashkey))
    if string(value) != "" && has == nil {
	db.Close()
	lock5.Unlock()
	return string(value),nil
    }

    db.Close()
    lock5.Unlock()
    return "",nil
}

func WriteLockoutInfoToLocalDB(hashkey string,value string) (bool,error) {
    if !IsInGroup() {
	return false,errors.New("it is not in group.")
    }

    if hashkey == "" || value == "" {
	return false,errors.New("param error in write lockout info to local db.")
    }

    lock5.Lock()
    path := GetDbDirForLockoutInfo()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============WriteLockoutInfoToLocalDB,create db fail.============")
	lock5.Unlock()
	return false,errors.New("create db fail.")
    }
    
    db.Put([]byte(hashkey),[]byte(value))
    db.Close()
    lock5.Unlock()
    return true,nil
}

//========
func ReadDccpAddrFromLocalDBByIndex(xvalue string,cointype string,index int) (string,error) {

    if xvalue == "" || cointype == "" || index < 0 {
	return "",errors.New("param error.")
    }

    lock4.Lock()
    path := GetDbDirForWriteDccpAddr()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============ReadDccpAddrFromLocalDBByIndex,create db fail.============")
	lock4.Unlock()
	return "",errors.New("create db fail.")
    }
    
    hash := crypto.Keccak256Hash([]byte(strings.ToLower(xvalue) + ":" + strings.ToLower(cointype))).Hex()
    value,has:= db.Get([]byte(hash))
    if string(value) != "" && has == nil {
	    v := strings.Split(string(value),":")
	    if len(v) < (index + 1) {
		db.Close()
		lock4.Unlock()
		return "",errors.New("has not dccpaddr in local DB.")
	    }

	    db.Close()
	    lock4.Unlock()
	    return v[index],nil
    }
	db.Close()
	lock4.Unlock()
	return "",errors.New("has not dccpaddr in local DB.")
}

func IsXValueAccountExsitDccpAddr(xvalue string,cointype string,dccpaddr string) (bool,string,error) {
    if xvalue == "" || cointype == "" {
	return false,"",errors.New("param error")
    }
    
    lock4.Lock()
    path := GetDbDirForWriteDccpAddr()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============IsXValueAccountExsitDccpAddr,create db fail.============")
	lock4.Unlock()
	return false,"",errors.New("create db fail.")
    }
    
    hash := crypto.Keccak256Hash([]byte(strings.ToLower(xvalue) + ":" + strings.ToLower(cointype))).Hex()
    if dccpaddr == "" {
	has,_ := db.Has([]byte(hash))
	if has == true {
		log.Debug("========IsXValueAccountExsitDccpAddr,has req dccpaddr.==============")
		value,_:= db.Get([]byte(hash))
		v := strings.Split(string(value),":")
		db.Close()
		lock4.Unlock()
		return true,string(v[0]),nil
	}

	log.Debug("========IsXValueAccountExsitDccpAddr,has not req dccpaddr.==============")
	db.Close()
	lock4.Unlock()
	return false,"",nil
    }
    
    value,has:= db.Get([]byte(hash))
    if has == nil && string(value) != "" {
	v := strings.Split(string(value),":")
	if len(v) < 1 {
	    log.Debug("========IsXValueAccountExsitDccpAddr,data error.==============")
	    db.Close()
	    lock4.Unlock()
	    return false,"",errors.New("data error.")
	}

	for _,item := range v {
	    if strings.EqualFold(item,dccpaddr) {
		log.Debug("========IsXValueAccountExsitDccpAddr,success get dccpaddr.==============")
		db.Close()
		lock4.Unlock()
		return true,dccpaddr,nil
	    }
	}
    }
   
    log.Debug("========IsXValueAccountExsitDccpAddr,fail get dccpaddr.==============")
    db.Close()
    lock4.Unlock()
    return false,"",nil

}

func WriteDccpAddrToLocalDB(xvalue string,cointype string,dccpaddr string) (bool,error) {
    if !IsInGroup() {
	return false,errors.New("it is not in group.")
    }

    if xvalue == "" || cointype == "" || dccpaddr == "" {
	return false,errors.New("param error in write dccpaddr to local db.")
    }

    lock4.Lock()
    path := GetDbDirForWriteDccpAddr()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============WriteDccpAddrToLocalDB,create db fail.============")
	lock4.Unlock()
	return false,errors.New("create db fail.")
    }
    
    hash := crypto.Keccak256Hash([]byte(strings.ToLower(xvalue) + ":" + strings.ToLower(cointype))).Hex()
    has,_ := db.Has([]byte(hash))
    if has != true {
	db.Put([]byte(hash),[]byte(dccpaddr))
	db.Close()
	lock4.Unlock()
	return true,nil
    }
    
    value,_:= db.Get([]byte(hash))
    v := string(value)
    v += ":"
    v += dccpaddr
    db.Put([]byte(hash),[]byte(v))
    db.Close()
    lock4.Unlock()
    return true,nil
}
//========

func ReadNodeInfoFromLocalDB(nodeinfo string) (string,error) {

    if nodeinfo == "" {
	return "",errors.New("param error in read nodeinfo from local db.")
    }

    lock3.Lock()
    path := GetDbDirForNodeInfoSave()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============ReadNodeInfoFromLocalDB,create db fail.============")
	lock3.Unlock()
	return "",errors.New("create db fail.")
    }
    
    value,has:= db.Get([]byte(nodeinfo))
    if string(value) != "" && has == nil {
	    db.Close()
	    lock3.Unlock()
	    return string(value),nil
    }
	db.Close()
	lock3.Unlock()
	return "",errors.New("has not nodeinfo in local DB.")
}

func IsNodeInfoExsitInLocalDB(nodeinfo string) (bool,error) {
    if nodeinfo == "" {
	return false,errors.New("param error in check local db by nodeinfo.")
    }
    
    lock3.Lock()
    path := GetDbDirForNodeInfoSave()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============IsNodeInfoExsitInLocalDB,create db fail.============")
	lock3.Unlock()
	return false,errors.New("create db fail.")
    }
    
    has,_ := db.Has([]byte(nodeinfo))
    if has == true {
	    db.Close()
	    lock3.Unlock()
	    return true,nil
    }

    db.Close()
    lock3.Unlock()
    return false,nil
}

func WriteNodeInfoToLocalDB(nodeinfo string,value string) (bool,error) {
    if !IsInGroup() {
	return false,errors.New("it is not in group.")
    }

    if nodeinfo == "" || value == "" {
	return false,errors.New("param error in write nodeinfo to local db.")
    }

    lock3.Lock()
    path := GetDbDirForNodeInfoSave()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============WriteNodeInfoToLocalDB,create db fail.============")
	lock3.Unlock()
	return false,errors.New("create db fail.")
    }
    
    db.Put([]byte(nodeinfo),[]byte(value))
    db.Close()
    lock3.Unlock()
    return true,nil
}

func IsHashkeyExsitInLocalDB(hashkey string) (bool,error) {
    if hashkey == "" {
	return false,errors.New("param error in check local db by hashkey.")
    }
    
    lock2.Lock()
    path := GetDbDirForLockin()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============IsHashkeyExsitInLocalDB,create db fail.============")
	lock2.Unlock()
	return false,errors.New("create db fail.")
    }
    
    has,_ := db.Has([]byte(hashkey))
    if has == true {
	    db.Close()
	    lock2.Unlock()
	    return true,nil
    }

	db.Close()
	lock2.Unlock()
	return false,nil
}

func WriteHashkeyToLocalDB(hashkey string,value string) (bool,error) {
    if !IsInGroup() {
	return false,errors.New("it is not in group.")
    }

    if hashkey == "" || value == "" {
	return false,errors.New("param error in write hashkey to local db.")
    }

    lock2.Lock()
    path := GetDbDirForLockin()
    db,_ := ethdb.NewLDBDatabase(path, 0, 0)
    if db == nil {
	log.Debug("==============WriteHashkeyToLocalDB,create db fail.============")
	lock2.Unlock()
	return false,errors.New("create db fail.")
    }
    
    db.Put([]byte(hashkey),[]byte(value))
    db.Close()
    lock2.Unlock()
    return true,nil
}

