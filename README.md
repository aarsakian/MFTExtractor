MFTExtractor
============

***A Parser  of Master File Table written  in go.

Still In progress but the majority of MFT attributes are parsed.



Usage information  type: MFTExtractor  -h
Dependencies gorp and go-sqlite3 
[go-sqlite3](https://github.com/mattn/go-sqlite3)
[gorp](https://github.com/coopernurse/gorp)


OUTPUT  (txt) is saved at the current directory as MFToutput.txt if -db argument is selected an mftsqlite file appears too.

************Explanation of OUTPUT***************************

MFT Entry number;Sequence Number;MFT flag;Standard Information;offset;Created Time;Accessed Time;Modified Time;MFT modified Time;File Name;offset;Access Time;Created Time;Modified Time;Filename;Parent MFT entry;Parent sequence number;Filename Flag;

Rest MFT attributes appear randomly: Some of them are described below (experimental)

Data Attr;True->Resident;

Index Root Type;size in clusters;nodeheader offset;nodeheader used entry list;

Attribute List:offset;Attribute Type;Attribute length;starting VCN;MFT record number;Attrribute file reference;Attribute Name;Attribute id;
			     
