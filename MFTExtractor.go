package main
import (
  "fmt"
  "os"
  "bytes"      
  "encoding/hex"
  "encoding/binary"
  "unicode/utf8"
  "unicode/utf16"
  "C"
  "strings"
  "time"
  "strconv"
  "log"
  "database/sql"
  _"github.com/mattn/go-sqlite3"
  "github.com/coopernurse/gorp"
  "flag"

  
   // "gob"//de-serialization 
   // "math"
)
  	

  



type MFTrecord struct{
  Signature string           //0-3
  UpdateSeqArrOffset uint16      //4-5      offset values are relative to the start of the entry.
  UpdateSeqArrSize uint16           //6-7
  Lsn  uint64 //8-15       logical File Sequence Number
  Seq uint16 //16-17   is incremented when the entry is either allocated or unallocated, determined by the OS.
  Linkcount uint16//18-19        how many directories have entries for this MFTentry
  Attr_off uint16//20-21       //first attr location
  Flags uint16 //22-23  //tells whether entry is used or not
  Size uint32 //24-27
  Alloc_size uint32//28-31
  Base_ref uint64//32-39
  Next_attrid  uint16//40-41 e.g. if it is 6 then there are attributes with 1 to 5
  F1 uint16//42-43
  Entry uint32//44-48                  ??
  Fncnt bool
  Data bool
  Bitmap bool
 // fixupArray add the        UpdateSeqArrOffset to find is location
  
  

  


}


type ATRrecordResident struct{
  Type  string    //        0-3                              type of attribute e.g. $DATA
  Len uint32  //4-8             length of attribute
  Res string//8
  Nlen  string
  Name_off uint16//name offset 10-12          relative to the start of attribute
  Flags uint16//12-14           //compressed, 
  Id uint16 //14-16 type of attribute 
  Ssize uint32 //16-20 size of resident attribute
  Soff uint16 //20-22 offset to content            soff+ssize=len
  Idxflag  uint16 //22-24
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use
  
}

type ATRrecordNoNResident struct{
  Type string    //        0-3                              type of attribute e.g. $DATA
  Len uint32  //4-8             length of attribute
  Res string//8 bool in original
  Nlen  string
  Name_off uint16//name offset 10-12          relative to the start of attribute
  Flags uint16//12-14           //compressed, 
  Id uint16 //14-16
    
  Start_vcn uint64 //16-24
  Last_vcn uint64 //24-32
  Run_off uint16 //32-24     offset to the start of the attribute
  Compusize uint16 //34-36
  F1 uint32    //36-40
  Alen uint64 //40-48
  NonRessize uint64 //48-56
  Initsize uint64 //56-64
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use

}

type WindowsTime struct{
  Stamp uint64
}

type FNAttribute struct{
  Par_ref uint64
  Par_seq uint16
  Crtime  WindowsTime
  Mtime WindowsTime//WindowsTime
  MFTmtime WindowsTime//WindowsTime
  Atime WindowsTime//WindowsTime
  Alloc_fsize uint64
  Real_fsize  uint64
  Flags  uint32//hidden Read Only? check Reparse
  Reparse uint32
  Nlen  uint8 //length of name
  Nspace uint8 //format of name
  Fname string//special string type without nulls
  HexFlag  bool
  UnicodeHack bool
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use
}



type ObjectID struct{//unique guid 
  Objid string //object id
  Orig_volid string	//volume id
  Orig_objid string //original objid
  Orig_domid  string// domain id
  EntryId uint32//foreing key 
  AttrId uint16 


}


type VolumeName struct{
  Name NoNull
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use
}



type IndexEntry struct{
  MFTfileref uint64//0-7
  Len uint16//8-9
  Contentlen uint16 //10-11
  Flags string //12-15
  
}

type IndexRoot struct{
  Type string//0-4 similar to FNA type
   // CollationSortingRule string
  Sizebytes uint32//8-12
  Sizeclusters uint8 //12-12
  nodeheader NodeHeader
}


type NodeHeader struct{
  OffsetEntryList uint32// 16-20 see 13.14
  OffsetEndUsedEntryList uint32 //20-24 where EntryList ends
  OffsetEndEntryListBuffer uint32//24-28
  Flags string

}
type IndexAllocation struct{
  Signature string //0-4
  FixupArrayOffset int16//4-6
  NumEntries int16//6-8
  LSN int64//8-16
  VCN int64 //16-24 where the record fits in the tree
  nodeheader NodeHeader
       

}
type AttributeList struct{//more than one MFT entry to store a file/directory its attributes
  Type string//        typeif 0-4    # 4
  len  uint16 //4-6 
  nlen uint8 //7unsigned char           # 1
  f1 uint8 //8-8               # 1
  start_vcn  uint64//8-16         # 8
  file_ref uint64 //16-22      # 6
  seq uint16 //       22-24    # 2
  id uint16    //     24-26   # 4
  name NoNull
            


}

type VolumeInfo struct{

  F1 uint64        //unused  
  Maj_ver string        
  Min_ver string
  Flags string //see table 13.22
  F2 uint32
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use


}


type SIAttribute struct{
  Crtime WindowsTime
  Mtime WindowsTime
  MFTmtime WindowsTime
  Atime WindowsTime
  Dos uint32
  Maxver uint32
  Ver uint32
  Class_id uint32
  Own_id uint32
  Sec_id uint32
  Quota uint64
  Usn uint64
  EntryId uint32//foreing key 
  AttrId uint16 //for DB use
}

func Bytereverse  (barray [] byte )([] byte ){//work with indexes
     //  fmt.Println("before",barray)
	for i, j := 0, len(barray)-1; i < j; i, j = i+1, j-1 {
	
	    barray[i], barray[j] = barray[j], barray[i]
	
	   
	}
      
      //  binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&val )
	//     fmt.Println("after",barray)
	   return  barray



}


func checkErr(err error, msg string) {
    if err != nil {
        log.Fatalln(msg, err)
    }
}

//
func initDb() *gorp.DbMap {
    // connect to db using standard Go database/sql API
    // use whatever database/sql driver you wish
    db, err := sql.Open("sqlite3", "./mft.sqlite")
    checkErr(err, "sql.Open failed")

    // construct a gorp DbMap
    dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}

    // add a table, setting the table name to 'posts' and
    // specifying that the Id property is an auto incrementing PK
    dbmap.AddTableWithName(MFTrecord{}, "MFTrecord").SetKeys(false, "Entry")
    dbmap.AddTableWithName(ATRrecordResident{},"ATRrecordResident").SetKeys(true,"AttrId")
    dbmap.AddTableWithName(ATRrecordNoNResident{},"ATRrecordNoNResident").SetKeys(true,"AttrId")
    dbmap.AddTableWithName(FNAttribute{},"FNAttribute")
    dbmap.AddTableWithName(SIAttribute{},"SIAttribute")
    dbmap.AddTableWithName(ObjectID{},"ObjectID")
    dbmap.AddTableWithName(VolumeInfo{},"VolumeInfo")
    dbmap.AddTableWithName(VolumeName{},"VolumeName")
    // create the table. in a production system you'd generally
    // use a migration tool, or create the tables via scripts
    err = dbmap.CreateTablesIfNotExists()
    checkErr(err, "Create tables failed")

    return dbmap
}


func Hexify(barray [] byte)(string){

  return hex.EncodeToString(barray)


}


func stringifyGuids(barray [] byte) string {
  s:= [] string {Hexify(barray[0:4]),Hexify(barray[4:6]),Hexify(barray[6:8]),Hexify(barray[8:10]),Hexify(barray[10:16])}
  return strings.Join(s,"-")
}


func readEndian (barray [] byte ) (val interface{}) {    
                    //conversion function
                 //fmt.Println("before conversion----------------",barray)           
     //fmt.Printf("len%d ",len(barray))
     
    switch len(barray) {
      case 8:
        var  vale uint64            
        binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&vale )	
        val=vale         
      case 6:
             
	      var  vale uint32   
	            buf := make([]byte, 6)
               binary.Read(bytes.NewBuffer(barray[:4])  ,binary.LittleEndian,&vale) 
			   var vale1 uint16
	      binary.Read(bytes.NewBuffer(barray[4:])  ,binary.LittleEndian,&vale1) 
	      binary.LittleEndian.PutUint32(buf[:4], vale)
	      binary.LittleEndian.PutUint16(buf[4:], vale1)
	      val ,_=binary.ReadUvarint(bytes.NewBuffer(buf))
           
		   
      case 4:
        var  vale uint32      
                            //   fmt.Println("barray",barray)
                                       binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&vale )      
        val=vale                                              
      case 2:     
    
               var  vale uint16                      
                  
                
                      binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&vale )         
                       //   fmt.Println("after conversion vale----------------",barray,vale) 
                           val=vale
			   
	 case 1:     
    
               var  vale uint8                      
                  
                
                      binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&vale )         
                    //      fmt.Println("after conversion vale----------------",barray,vale) 
                           val=vale    		   
                                      
        default://best it would be nil 
           var  vale uint64    
                 
                 binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&vale )          
                   val=vale    
            }                     
              
          //     b:=[]byte{0x18,0x2d}        
                
               //    fmt.Println("after conversion val",val) 
      return val
}

    
func (winTime * WindowsTime) convertToIsoTime() string {//receiver winTime struct
    var localTime string
    if (winTime.Stamp != 0) {
       // t:=math.Pow((uint64(winTime.high)*2),32) + uint64(winTime.low)
	x:=winTime.Stamp/10000000 - 116444736*1e2
	unixtime:= time.Unix(int64(x),0).UTC()
	localTime=unixtime.Format("02-01-2006 15:04:05")
       // fmt.Println("time",unixtime.Format("02-01-2006 15:04:05"))
      
    }else{
     localTime="---"
    }
  return localTime
}






func readEndianFloat (barray [] byte) (val uint64) {    
               
 //    fmt.Printf("len%d ",len(barray))
            
     
        
      binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&val )
      return val 
}


func readEndianInt(barray [] byte) ( uint64) {    
               
     
   
      //fmt.Println("------",barray,barray[len(barray)-1])
      var sum uint64
      sum=0
      for index,val :=range barray {
	sum+=uint64(val)<<uint(index*8)
      
	//fmt.Println(sum)
      }
        
    
      return sum
}


type NoNull string;

func readEndianString (barray [] byte) (val [] byte) {    
               
  
            
     
        
      binary.Read(bytes.NewBuffer(barray)  ,binary.LittleEndian,&val )
   
      return val 
}

func ProcessRunList (val byte)(string,uint64,uint64) {
  var (
    ClusterLen string
    ClusterOffs string
    clusteroffs uint64
	clusterlen uint64
    )

  //fmt.Println("BARRAY ",barray)
 
 
    
    val1:=(fmt.Sprintf("%x",val))
  //  fmt.Printf("Val %s",val1)
    if (len(val1)==2){//little Endian onrder on strigns damn type
  
      ClusterOffs =val1[0:1]
      ClusterLen=val1[1:2]

      clusterlen,_=strconv.ParseUint(ClusterLen,8,8)
      clusteroffs,_=strconv.ParseUint(ClusterOffs,8,8)
   
    

    }else{
        ClusterOffs="0"
	ClusterLen="0"
    }
 
    
 
  //  fmt.Printf("Cluster located at %s and lenght %s\n",ClusterOffs, ClusterLen)
    return val1,clusteroffs,clusterlen
 
  
}

func DecodeUTF16(b[]byte )string{
    utf:=make([]uint16,(len(b)+(2-1))/2)//2 bytes for one char?
    for i:=0; i+(2-1)<len(b);i+=2{
      utf[i/2]=binary.LittleEndian.Uint16(b[i:])
    }
    if len(b)/2<len(utf){
      utf[len(utf)-1]=utf8.RuneError
    }
    return string(utf16.Decode(utf))


}
func (str  *NoNull) PrintNulls()  string{
  var newstr []string
  for _,v :=range *str{
    if v!=0{
    
      newstr = append(newstr,string(v))
     
    }
  }
  return  strings.Join(newstr,"")
}


func main() {
  dbmap := initDb()
  defer dbmap.Db.Close()
  
  save2DB := flag.Bool("db", false, "bool if set an sqlite file will be created, each table will corresponed to an MFT attribute")
  inputfile :=  flag.String("MFT","MFT file", "absolute path to the MFT file")		
  flag.Parse()//ready to parse
		
  err := dbmap.TruncateTables()
  checkErr(err, "TruncateTables failed")
  IndexEntryFlags := map[string]string{
    "00000001":"Child Node exists",
    "00000002":"Last Entry in list",
  }
 
  AttrTypes := map[string]string{
     "00000010":"Standard Information","00000020":"Attribute List","00000030":"File Name","00000040":"Object ID",
     "00000050":"Security Descriptor","00000060":"Volume Name","00000070":"Volume Information","00000080":"Data",
     "00000090":"Index Root","000000A0":"Index Allocation","000000B0":"BitMap","000000C0":"Reparse Point",
  }
  
  Flags:=map[uint32]string{
    1:"Read Only",2:"Hidden",4:"System",32:"Archive",64:"Device",128:"Normal",
    256:"Temporary",512:"Sparse",1024:"Reparse Point",2048:"Compressed",4096:"Offline",
    8192:"Not Indexed",16384:"Encrypted",
  }
  
  MFTflags:=map[uint16]string{
    0:"File Unallocted",1:"File Allocated",2:"Folder Unalloc",3:"Folder Allocated",
  }
  fmt.Println(*inputfile,os.Args[1])
  file, err := os.Open(*inputfile)//,"F:\\3022_21_2524\\Registry\\$MFT"

  if err != nil {
    // handle the error here
     fmt.Printf("err %s for reading the MFT ",err)
    return
  }
 
 
 
  // get the file size
  fsize, err := file.Stat()    //file descriptor
  if err != nil {
    return
  }
  // read the file     C:\\DEELAB\\GoProgs\\MFToutput.txt
  file1, err:= os.OpenFile("MFToutput.txt",os.O_RDWR|os.O_CREATE,0666)

  if err != nil {
    // handle the error here
     fmt.Printf("err %s",err)
    return
  }
  defer file.Close()
  defer file1.Close()
  


  
  var k=0
  _,err1:=file1.WriteString("FILE SIZE------------------------------------------------------------"+fmt.Sprintf("%d",fsize.Size())+string(10))
  if err1 != nil {
      // handle the error here
    fmt.Printf("err %s\n",err)	
    return
  }
  for i:=0 ;i<=int(fsize.Size());i+=1024 {
    //if (i<=2048){
    bs := make([]byte, 1024)   //new byte array of length MFT entry
       
    _, err := file.ReadAt(bs,int64(i))                              
              // fmt.Printf("\n I read %s and out is %d\n",hex.Dump(bs[20:22]), readEndian(bs[20:22]).(uint16)) 
    if err != nil {
      fmt.Printf("err edw --->%s",err)
      return
    }  
 

    if string(bs[:4])=="FILE" {
                      
            
      r:= MFTrecord{string(bs[:4]),readEndian(bs[4:6]).(uint16),readEndian(bs[6:8]).(uint16),readEndian(bs[8:16]).(uint64), readEndian(bs[16:18]).(uint16),
         readEndian(bs[18:20]).(uint16),  readEndian(bs[20:22]).(uint16),readEndian(bs[22:24]).(uint16),readEndian(bs[24:28]).(uint32),readEndian(bs[28:32]).(uint32),
          readEndian(bs[32:40]).(uint64),readEndian(bs[40:42]).(uint16),readEndian(bs[42:44]).(uint16),readEndian(bs[44:48]).(uint32),false,false,false }                  //check to see if returning implements the corresponding interface
      
      if  *save2DB   {        
	dbmap.Insert(&r)
	checkErr(err, "Insert failed")
      }
	
          
      _,err1:=file1.WriteString(fmt.Sprintf("\n%d;%d;%s",r.Entry,r.Seq ,MFTflags[r.Flags]))
      if err1 != nil {
		// handle the error here
	   // fmt.Printf("err %s\n",err)	
	return
      }

                  

     // err = dbmap.Insert(&r)
     // checkErr(err, "Insert failed")
    
   

    



      if r.Signature != "BAAD" {//enty signature if BAAD error value
			
	  
	ReadPtr := r.Attr_off //first attribute
          //  fmt.Println("Resident? ",Bytereverse(bs[ReadPtr+8:ReadPtr+9]))
	for ReadPtr < 1024{

	    
	  if hex.EncodeToString(bs[ReadPtr:ReadPtr+4])== "ffffffff"         {               //End of attributes
            break 
          }
	  
	  
            // fmt.Printf("Type %s Residnet  Attr %b Endian \n",hex.EncodeToString(bs[ReadPtr:ReadPtr+4]),readEndianString(bs[ReadPtr:ReadPtr+4]))
          if Hexify(bs[ReadPtr+8:ReadPtr+9])=="00"  {     //Resident Attribute
	    ATR :=  ATRrecordResident{Hexify(Bytereverse(bs[ReadPtr:ReadPtr+4])),readEndian(bs[ReadPtr+4:ReadPtr+8]).(uint32),string(bs[ReadPtr+8:ReadPtr+9]),string(bs[ReadPtr+9:ReadPtr+10]),
                   readEndian(bs[ReadPtr+10:ReadPtr+12]).(uint16), readEndian(bs[ReadPtr+12:ReadPtr+14]).(uint16), readEndian(bs[ReadPtr+14:ReadPtr+16]).(uint16),
                   readEndian(bs[ReadPtr+16:ReadPtr+20]).(uint32), readEndian(bs[ReadPtr+20:ReadPtr+22]).(uint16),readEndian(bs[ReadPtr+22:ReadPtr+24]).(uint16),
		   r.Entry,0}//start from offset till end	     
	    if  *save2DB   {                           
	      dbmap.Insert(&ATR)
	      checkErr(err, "Insert failed")
	    }
	         //   fmt.Printf("Resident type %s where data length %d and starts at %d ,Attribute length %d  equal>%b \n",ATR.Type,ATR.ssize,ATR.Soff,ATR.Len,uint32(ATR.Soff)+ATR.ssize==ATR.Len)
	    s := strings.Join([] string {";",AttrTypes[ATR.Type]},"")
	    _,err:=file1.WriteString(s)
	    if err != nil {
			    // handle the error here
	      fmt.Printf("err %s\n",err)	
	      return
	    }
	    if  ATR.Type == "ffffffff" {             // End of attributes
	      break 
					     
	    } else if   ATR.Type== "00000030"       {          // File name
	      Crtime:=WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+8:ReadPtr+ATR.Soff+16]).(uint64)}
	      Mtime :=WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+24]).(uint64)}
	      MFTTime:=WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+24:ReadPtr+ATR.Soff+32]).(uint64)}
	      Atime := WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+32:ReadPtr+ATR.Soff+40]).(uint64)}
	      fnattr := FNAttribute{readEndian(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+6]).(uint64),
		                     readEndian(bs[ReadPtr+ATR.Soff+6:ReadPtr+ATR.Soff+8]).(uint16),
                         Crtime,
			 Mtime,
			MFTTime,
                        Atime,
		         readEndian(bs[ReadPtr+ATR.Soff+40:ReadPtr+ATR.Soff+48]).(uint64),readEndian(bs[ReadPtr+ATR.Soff+48:ReadPtr+ATR.Soff+56]).(uint64),
                         readEndian(bs[ReadPtr+ATR.Soff+56:ReadPtr+ATR.Soff+60]).(uint32),readEndian(bs[ReadPtr+ATR.Soff+60:ReadPtr+ATR.Soff+64]).(uint32),
			 readEndian(bs[ReadPtr+ATR.Soff+64:ReadPtr+ATR.Soff+65]).(uint8),
			  readEndian(bs[ReadPtr+ATR.Soff+65:ReadPtr+ATR.Soff+66]).(uint8),
			 DecodeUTF16(bs[ReadPtr+ATR.Soff+66:ReadPtr+ATR.Soff+66+2*uint16(readEndian(bs[ReadPtr+ATR.Soff+64:ReadPtr+ATR.Soff+65]).(uint8))]),
			 false,false,r.Entry,0}
			  //  fmt.Println("\nFNA ",bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+65],bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+6],readEndian(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+6]).(uint64),
				//	"PAREF",fnattr.par_ref,"SQ",fnattr.fname,"FLAG",fnattr.flags)
			   //   fmt.Printf("time Mod %s time Accessed %s time Created %s Filename %s\n ", fnattr.atime.convertToIsoTime(),fnattr.crtime.convertToIsoTime(),fnattr.mtime.convertToIsoTime(),fnattr.fname )
			  //    fmt.Println(strings.TrimSpace(string(bs[ReadPtr+ATR.Soff+66:ReadPtr+ATR.Soff+66+2*uint16(readEndian(bs[ReadPtr+ATR.Soff+64:ReadPtr+ATR.Soff+65]).(uint8))])))
	      if  *save2DB   {         
		dbmap.Insert(&fnattr)
		checkErr(err, "Insert failed")	
	      }
	      s := strings.Join([] string {fmt.Sprintf(";%d",ReadPtr+ATR.Soff), ";", fnattr.Atime.convertToIsoTime(),";",fnattr.Crtime.convertToIsoTime(),
			       ";",fnattr.Mtime.convertToIsoTime(), ";",fnattr.Fname, fmt.Sprintf(";%d;%d;%s",fnattr.Par_ref,fnattr.Par_seq, Flags[fnattr.Flags])},"")
				     
			    
			      
	      _,err:=file1.WriteString(s)//(string(10)) breaks line
	      if err != nil {
		 // handle the error here
		fmt.Printf("err %s\n",err)	
		return
	      }
            } else if  ATR.Type == "00000080" {
	      r.Data=true
	      _,err:=file1.WriteString(";"+strconv.FormatBool(r.Data))
	      if err != nil {
		     // handle the error here
		fmt.Printf("err %s\n",err)	
		  return
	      }	
			    
	    } else if ATR.Type == "00000040" {
	      objectattr := ObjectID{stringifyGuids(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+16]),
		                              stringifyGuids(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+32]),
		                              stringifyGuids(bs[ReadPtr+ATR.Soff+32:ReadPtr+ATR.Soff+48]),
					      stringifyGuids(bs[ReadPtr+ATR.Soff+48:ReadPtr+ATR.Soff+64]),
					      r.Entry,0}
		      // fmt.Println("file unique id ",objectattr.objid)
	      if  *save2DB   {        
		dbmap.Insert(&objectattr)
		checkErr(err, "Insert failed")
	      }
	      s := [] string  {";",objectattr.Objid}
	      _,err:=file1.WriteString(strings.Join(s," "))
	      if err != nil {
			    // handle the error here
		fmt.Printf("err %s\n",err)	
		 return
	      }  
	    } else if ATR.Type == "00000020" {//Attribute List
	    //  runlist:=bs[ReadPtr+ATR.Soff:uint32(ReadPtr)+ATR.Len]
	      var attrLen uint16
	      attrLen=0
	      for  ATR.Soff+26+attrLen< uint16(ATR.Len){
		//fmt.Println("TEST",len(runlist),26+attrLen+ATR.Soff, uint16(ATR.Len))
		attrList:=AttributeList{Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+attrLen:ReadPtr+ATR.Soff+4+attrLen])),
			         readEndian(bs[ReadPtr+ATR.Soff+4+attrLen:ReadPtr+ATR.Soff+6+attrLen]).(uint16),
			         readEndian(bs[ReadPtr+ATR.Soff+6+attrLen:ReadPtr+ATR.Soff+7+attrLen]).(uint8),readEndian(bs[ReadPtr+ATR.Soff+7:ReadPtr+ATR.Soff+8]).(uint8),
				 readEndian(bs[ReadPtr+ATR.Soff+8+attrLen:ReadPtr+ATR.Soff+16+attrLen]).(uint64),
				 readEndian(bs[ReadPtr+ATR.Soff+16+attrLen:ReadPtr+ATR.Soff+22+attrLen]).(uint64),readEndian(bs[ReadPtr+ATR.Soff+22:ReadPtr+ATR.Soff+24]).(uint16),
				 readEndian(bs[ReadPtr+ATR.Soff+24+attrLen:ReadPtr+ATR.Soff+26+attrLen]).(uint16),
				  NoNull(bs[ReadPtr+ATR.Soff+26+attrLen:ReadPtr+ATR.Soff+32+attrLen])}
			  //     fmt.Println("START VCN",attrList.start_vcn)
			       
			          
		s := [] string  {"Type of Attr in Run list", fmt.Sprintf("Attribute starts at %d",ReadPtr),
			      AttrTypes[attrList.Type],fmt.Sprintf("length %d ",attrList.len),fmt.Sprintf("start VCN %d ",attrList.start_vcn),
			     "MFT Record Number",fmt.Sprintf("%d Name %s",attrList.file_ref,attrList.name),
			    "Attribute id ",fmt.Sprintf("%d ",attrList.id),string(10)}
		_,err:=file1.WriteString(strings.Join(s," "))
		if err != nil {
			    // handle the error here
		  fmt.Printf("err %s\n",err)	
		  return
		}
			   
			   //   runlist=bs[ReadPtr+ATR.Soff+attrList.len:uint32(ReadPtr)+ATR.Len]
		attrLen+=attrList.len
			     
			         
	      }   
	    } else  if   ATR.Type== "000000b0"     {//BITMAP
	      r.Bitmap=true
		   
	    } else  if   ATR.Type== "00000060"     {//Volume Name
	      volname:= VolumeName{NoNull(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+16]),r.Entry,0}
	      if  *save2DB   {        	    
		dbmap.Insert(&volname)
		checkErr(err, "Insert failed")
	      }
	      
	      s := [] string  {";",volname.Name.PrintNulls()}
	      _,err:=file1.WriteString(strings.Join(s,"s"))
	      if err != nil {
		// handle the error here
		fmt.Printf("err %s\n",err)	
		return
	      }  
	    } else  if   ATR.Type== "00000070"     {//Volume Info
	      volinfo := VolumeInfo{readEndian(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+8]).(uint64),
			  Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+8:ReadPtr+ATR.Soff+9])),
			  Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+9:ReadPtr+ATR.Soff+10])),
			  Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+10:ReadPtr+ATR.Soff+12])),
			  readEndian(bs[ReadPtr+ATR.Soff+12:ReadPtr+ATR.Soff+16]).(uint32),
			  r.Entry,0}
	      if  *save2DB   {        
	        dbmap.Insert(&volinfo)
		checkErr(err, "Insert failed")
	      }
	      
	      s := [] string  {"Vol Info flags",volinfo.Flags,string(10)}
	      _,err:=file1.WriteString(strings.Join(s," "))
	      if err != nil {
			    // handle the error here
		fmt.Printf("err %s\n",err)	
		return
	      }  
	    } else  if   ATR.Type== "00000090"     {//Index Root
	
	
	      nodeheader:= NodeHeader {readEndian(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+20]).(uint32),readEndian(bs[ReadPtr+ATR.Soff+20:ReadPtr+ATR.Soff+24]).(uint32),
			    readEndian(bs[ReadPtr+ATR.Soff+24:ReadPtr+ATR.Soff+28]).(uint32),Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+28:ReadPtr+ATR.Soff+32]))}
	      idxroot := IndexRoot{string(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+4]),readEndian(bs[ReadPtr+ATR.Soff+8:ReadPtr+ATR.Soff+12]).(uint32),
			 readEndian(bs[ReadPtr+ATR.Soff+12:ReadPtr+ATR.Soff+13]).(uint8),nodeheader}
			 
	      idxentry :=  IndexEntry{readEndian(bs[ReadPtr+ATR.Soff+32:ReadPtr+ATR.Soff+40]).(uint64),readEndian(bs[ReadPtr+ATR.Soff+40:ReadPtr+ATR.Soff+42]).(uint16),
	              readEndian(bs[ReadPtr+ATR.Soff+42:ReadPtr+ATR.Soff+44]).(uint16), Hexify(Bytereverse(bs[ReadPtr+ATR.Soff+44:ReadPtr+ATR.Soff+48]))}	 
			// 
	      s := [] string  {idxroot.Type,";",fmt.Sprintf(";%d",idxroot.Sizeclusters),";",fmt.Sprintf("%d;",16+idxroot.nodeheader.OffsetEntryList),
		      fmt.Sprintf(";%d",16+idxroot.nodeheader.OffsetEndUsedEntryList),fmt.Sprintf("allocated ends at %d",16+idxroot.nodeheader.OffsetEndEntryListBuffer),
		      fmt.Sprintf("MFT entry%d ",idxentry.MFTfileref),"FLags",IndexEntryFlags[idxentry.Flags]}
		      //fmt.Sprintf("%x",bs[uint32(ReadPtr)+uint32(ATR.Soff)+32:uint32(ReadPtr)+uint32(ATR.Soff)+16+idxroot.nodeheader.OffsetEndEntryListBuffer]
				 
	      _,err:=file1.WriteString(strings.Join(s," "))
	      if err != nil {
			    // handle the error here
		fmt.Printf("err %s\n",err)	
		return
	      }  
	    }  else  if   ATR.Type== "00000010"       {
	      startpoint:=ReadPtr+ATR.Soff
	      siattr:=SIAttribute{WindowsTime{readEndian(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+8]).(uint64)},
			                      WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+8:ReadPtr+ATR.Soff+16]).(uint64)},
					      WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+24]).(uint64)},
			                      WindowsTime{readEndian(bs[ReadPtr+ATR.Soff+24:ReadPtr+ATR.Soff+32]).(uint64)},
			                    readEndian(bs[startpoint+32:startpoint+36]).(uint32),
			       readEndian(bs[startpoint+36:startpoint+40]).(uint32),readEndian(bs[startpoint+40:startpoint+44]).(uint32),readEndian(bs[startpoint+44:startpoint+48]).(uint32),
			       readEndian(bs[startpoint+48:startpoint+52]).(uint32),readEndian(bs[startpoint+52:startpoint+56]).(uint32),readEndian(bs[startpoint+56:startpoint+64]).(uint64),
			       readEndian(bs[startpoint+64:startpoint+72]).(uint64),r.Entry,0}
	      if  *save2DB   {        
		dbmap.Insert(&siattr)
		checkErr(err, "Insert failed")	
	      }
	      s := [] string {fmt.Sprintf(";%d", startpoint),";",siattr.Crtime.convertToIsoTime(),
		             ";",siattr.Atime.convertToIsoTime(),";",siattr.Mtime.convertToIsoTime(),";",siattr.MFTmtime.convertToIsoTime()}
	      _,err:=file1.WriteString(strings.Join(s,""))
	      if err != nil {
			    // handle the error here
		fmt.Printf("err %s\n",err)	
		return
	      }   
			    
			    
	    }
		   
	    if ATR.Len > 0   {
	      ReadPtr = ReadPtr + uint16(ATR.Len)
	    }		                                                    
	  }  else {  //NoN Resident Attribute
            ATR :=  ATRrecordNoNResident{Hexify(Bytereverse(bs[ReadPtr:ReadPtr+4])),readEndian(bs[ReadPtr+4:ReadPtr+8]).(uint32),string(bs[ReadPtr+8:ReadPtr+9]),string(bs[ReadPtr+9:ReadPtr+10]),
                 readEndian(bs[ReadPtr+10:ReadPtr+12]).(uint16), readEndian(bs[ReadPtr+12:ReadPtr+14]).(uint16), readEndian(bs[ReadPtr+14:ReadPtr+16]).(uint16),
                  readEndian(bs[ReadPtr+16:ReadPtr+24]).(uint64), readEndian(bs[ReadPtr+24:ReadPtr+32]).(uint64),readEndian(bs[ReadPtr+32:ReadPtr+34]).(uint16),
                 readEndian(bs[ReadPtr+34:ReadPtr+36]).(uint16),  readEndian(bs[ReadPtr+36:ReadPtr+40]).(uint32),readEndian(bs[ReadPtr+40:ReadPtr+48]).(uint64),
                  readEndian(bs[ReadPtr+48:ReadPtr+56]).(uint64),  readEndian(bs[ReadPtr+56:ReadPtr+64]).(uint64),
		  r.Entry,0}//start from offset till end
				     
			      //        fmt.Println("NON Resident type ",ATR.Type,ATR.Len,ReadPtr)
			//  buffer:=[] byte{}// a slice
	    if  *save2DB   {        	
	      dbmap.Insert(&ATR)
	      checkErr(err, "Insert failed")
	    }
	    
	    s := [] string {";", AttrTypes[ATR.Type],fmt.Sprintf(";%d",ReadPtr),";false",fmt.Sprintf(";%d;%d",ATR.Start_vcn,ATR.Last_vcn)}
	    _,err:=file1.WriteString(strings.Join(s,""))
	    if err != nil {
			    // handle the error here
	      fmt.Printf("err %s\n",err)	
	      return
	    }	           
            if  ATR.Type == "ffffffff" {             // End of attributes
		  break 
					     
	    } else if  ATR.Type == "00000080" {
	      r.Data=true
	      if uint32(ReadPtr)+ATR.Len<=1024{
		runlist:=bs[ReadPtr+ATR.Run_off:uint32(ReadPtr)+ATR.Len]
		var Clusters uint64
		Clusters=0
	                   // fmt.Printf("LEN %d RUNLIST %x\n" ,len(runlist),runlist)
		for index, val := range runlist{
		  _,ClusterOffs,ClusterLen :=ProcessRunList(val)
			   		
		  if ClusterLen!=0 && ClusterOffs!=0{
			      //   fmt.Println("reading from",uint64(ReadPtr)+uint64(ATR.Run_off)+uint64(index),"ews ",
				  //     uint64(ReadPtr)+uint64(ATR.Run_off)+uint64(index)+ClusterLen+ClusterOffs,
				    //    "Atr starts at",ReadPtr+ATR.Run_off,"ATR LEN",uint16(ATR.Len),"reading at",uint64(ReadPtr)+uint64(ATR.Run_off)+uint64(index)+ClusterLen+ClusterOffs)
				
		    ClustersLen:= readEndianInt(bs[uint64(ReadPtr)+uint64(ATR.Run_off)+1:uint64(ReadPtr)+uint64(ATR.Run_off)+ClusterLen+1])
				
		    ClustersOff:=readEndianInt(bs[uint64(ReadPtr)+uint64(ATR.Run_off)+ClusterLen+1:uint64(ReadPtr)+uint64(ATR.Run_off)+ClusterLen+ClusterOffs+1])
				    //  fmt.Printf("len of %d clusterlen %d and clust %d clustoff %d came from %x \n",ClusterLen,ClustersLen,ClusterOffs,ClustersOff,val)
				//readEndianInt(bs[uint64(ReadPtr)+uint64(ATR.Run_off)+1:uint64(ReadPtr)+uint64(ATR.Run_off)+ClusterLen+1]))
		    s := [] string {  fmt.Sprintf(";%d;%d",ClustersOff,ClustersLen)}
		    _,err:=file1.WriteString(strings.Join(s," "))
		    if err != nil {
				  // handle the error here
		      fmt.Printf("err %s\n",err)	
		      return
		    }
				
				 //fmt.Println("lenght of runlist",len(runlist),"cluster len" ,ClusterLen+ClusterOffs,"runlist",runlist)
		    if ClusterLen+ClusterOffs<uint64(len(runlist)){
		      runlist=bs[uint64(ReadPtr)+uint64(ATR.Run_off)+uint64(index)+Clusters+ClusterLen+ClusterOffs:uint32(ReadPtr)+ATR.Len]
		      Clusters+=ClusterLen+ClusterOffs
		    } else {
		       break
		    }
		  } else {
		    break
		  }	       
		}
	      }
			 
			 //s := [] string {fmt.Sprintf("Start VCN %d END VCN %d",ATR.start_vcn,ATR.last_vcn ), string(10)}
			 // _,err:=file1.WriteString(strings.Join(s," "))
		         //  if err != nil {
			    // handle the error here
			 //   fmt.Printf("err %s\n",err)	
			 //     return
			  //  }	    	    
	    }	  
			    
		     /*else if  ATR.Type == "000000a0" {//Index Allcation
			     nodeheader := NodeHeader {readEndian(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+20]).(uint32),readEndian(bs[ReadPtr+ATR.Soff+20:ReadPtr+ATR.Soff+24]).(uint32),
			    readEndian(bs[ReadPtr+ATR.Soff+24:ReadPtr+ATR.Soff+28]).(uint32)}
		        idxall := IndexAllocation{string(bs[ReadPtr+ATR.Soff:ReadPtr+ATR.Soff+4]),readEndian(bs[ReadPtr+ATR.Soff+4:ReadPtr+ATR.Soff+6]).(uint16),readEndian(bs[ReadPtr+ATR.Soff+6:ReadPtr+ATR.Soff+8]).(uint16),
			    readEndian(bs[ReadPtr+ATR.Soff+16:ReadPtr+ATR.Soff+24]).(uint64), nodeheader}
			 
			 s := [] string  {"Index Allocation Type ",idxall.Type,fmt.Sprintf("VCN %d  ",idxall.VCN),"Index entry start",fmt.Sprintf("%d",idxall.nodeheader.OffsetEntryList),
			        fmt.Sprintf(" used portion ends at %d",idxall.nodeheader.OffsetEndUsedEntryList),fmt.Sprintf("allocated ends at %d",idxall.nodeheader.OffsetEndEntryListBuffer)  ,string(10)}
			  _,err:=file1.WriteString(strings.Join(s," "))
			  if err != nil {
			    // handle the error here
			    fmt.Printf("err %s\n",err)	
			      return
			    }  
			    
		   }*/
	    if ATR.Len > 0   {
	      ReadPtr = ReadPtr + uint16(ATR.Len)
	    }
              
	      
	      
	      
	      
	      
	      
	      
	      
	      
	  }//ends non resident
				
				     
				          
				  
             
        } //ends while

      }//ends if
      k++ 
    }
  }   //ends for
}
