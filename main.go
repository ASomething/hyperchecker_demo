package main

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"time"
	"encoding/json"
)

type ApartmentRegister struct {
}

type Renter struct {
	name    string         // 名
	surname string		   // 姓
	movedIn time.Time
}

type Block struct {
	id       string
	street   string
	number   string
	renters  []Renter
	nOfRooms string
}

//cache of blocks id  ---> 存储区块的id，用于判断一个区块是否已经存在
var blocks map[string]bool   // 声明变量blocks

func createId(street string, number string) string {
	return fmt.Sprintf("%s%d", street, number)
}

//retrieve a block on the ledger
func getBlock(stub shim.ChaincodeStubInterface, key string) (*Block, error) {
	var block Block
	block_marshalled, err := stub.GetState(key)
	err = json.Unmarshal(block_marshalled, &block)
	return &block, err
}

//save a block on the ledger
func putBlock(stub shim.ChaincodeStubInterface, key string, block *Block) error {
	block_marshalled, _ := json.Marshal(*block)
	return stub.PutState(key, block_marshalled)
}

//returns information about a renter
func queryRenter(stub shim.ChaincodeStubInterface, street string, number string, name string) peer.Response {
	id := createId(street, number)
	if !blocks[id] {
		return shim.Error(fmt.Sprintf("No block %s registered", id))
	}
	block, _ := getBlock(stub, id)
	var renter *Renter
	for _, r := range block.renters {
		if r.name == name {
			renter = &r
		}
	}
	if renter == nil {
		return shim.Error(fmt.Sprintf("Could not find renter %s in block %s", name, id))
	} else {
		renter_marshalled, _ := json.Marshal(*renter)
		return shim.Success(renter_marshalled)
	}
}

//registers a new renter in an apartmentblock   ---> 在一个区块中注册一个新租客
func registerNewRenter(stub shim.ChaincodeStubInterface, street string, number string, name string, surname string) peer.Response {
	id := createId(street, number)
	if !blocks[id] {
		return shim.Error(fmt.Sprintf("No block %s registered", id))
	}
	block, err := getBlock(stub, id)
	if err != nil {
		return shim.Error(fmt.Sprintf("could not retrieve %s", id))
	}

	now := time.Now()
	renter := Renter{
		name:    name,
		surname: surname,
		movedIn: now,
	}

	block.renters = append(block.renters, renter)

	block_marshalled, _ := json.Marshal(*block)
	err = stub.PutState(id, block_marshalled)
	if err != nil {
		return shim.Error(fmt.Sprintf("could not update %s", id))
	}
	block_marshalled, err = stub.GetState(id)
	json.Unmarshal(block_marshalled, block)
	return shim.Success([]byte(fmt.Sprintf("Block %s has now %d renters", id, len(block.renters))))
}

//creates a new block
func newBlock(stub shim.ChaincodeStubInterface, street string, number string, nOfRooms string) peer.Response {
	id := createId(street, number)
	if blocks[id] {
		return shim.Error(fmt.Sprintf("Block %s already exists", id))
	}
	blocks[id] = true

	block := new(Block)
	block.id = id
	block.street = street
	block.number = number
	block.nOfRooms = nOfRooms
	putBlock(stub, id, block)

	blocks_marshalled, _ := json.Marshal(blocks)
	stub.PutState("blocksIdCache", blocks_marshalled)

	return shim.Success([]byte(fmt.Sprintf("Successfully created block %s.", id)))
}

//returns the number of blocks ---> 打印当前区块高度
func blocksCount() peer.Response {
	count := len(blocks)
	return shim.Success([]byte(fmt.Sprintf("%d blocks found", count)))
}

//returns the number of renters in a specific block  ---> 返回区块中renters字段的长度(即租客的数量)
func rentersCount(stub shim.ChaincodeStubInterface, street string, number string) peer.Response {
	id := createId(street, number)
	block, err := getBlock(stub, id)
	if block == nil || err != nil {
		return shim.Error(fmt.Sprintf("could not retrieve %s %d", street, number))
	}

	return shim.Success([]byte(fmt.Sprintf("%d renters in %s found", len(block.renters), block.id)))
}

//Finds an apartmentblock whithout any renters    ---> 找到没有租客的区块
func findEmptyBlock(stub shim.ChaincodeStubInterface) peer.Response {
	for id, _ := range blocks {
		block, err := getBlock(stub, id)
		if err != nil {
			shim.Error(fmt.Sprintf("Could not find block %s", id))
		}
		if len(block.renters) == 0 {
			block_marshalled, err := json.Marshal(*block)
			if err != nil {
				return shim.Error("Could not marshal block.")
			}
			return shim.Success(block_marshalled)
		}
	}
	return shim.Error("No blocks empty")
}

//Initialisation of the Chaincode  ---> 初始化链码
func (m *ApartmentRegister) Init(stub shim.ChaincodeStubInterface) peer.Response {
	blocks = make(map[string]bool)        // // 为变量blocks开辟空间
	return shim.Success([]byte("Successfully initialized Chaincode."))
}

//Entry Point of an invocation    ---> 调用入口
func (m *ApartmentRegister) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, para := stub.GetFunctionAndParameters()

	switch(function) {
	case "queryRenter":
		if len(para) < 3 {     // 嗯。。。stub没有算入
			return shim.Error("not enough arguments for queryRenter. 3 required")
		} else {
			return queryRenter(stub, para[0], para[1], para[2])
		}
	case "registerRenter":
		if len(para) < 3 {
			return shim.Error("not enough arguments for registerRenter. 4 required")
		} else {
			return registerNewRenter(stub, para[0], para[1], para[2], para[3])
		}
	case "newBlock":
		return newBlock(stub, para[0], para[1], para[2])
	case "blocksCount":
		return blocksCount()
	case "rentersCount":
		if len(para) < 2 {
			return shim.Error("not enough arguments for rentersCount. 2 required")
		} else {
			return rentersCount(stub, para[0], para[1])
		}
	case "findEmptyBlock":
		return findEmptyBlock(stub)
	}
	return shim.Error(fmt.Sprintf("No function %s implemented", function))    // 函数匹配失败
}

func main() {
	if err : = shim.Start(new(ApartmentRegister)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}
