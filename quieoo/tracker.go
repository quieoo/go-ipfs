package quieoo

import (
	"errors"
	"fmt"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log"
	"github.com/multiformats/go-multihash"
	"sync"
	"time"
)

type FTracker struct {

	WantBlocksT time.Time//the time of wantBlocks
	FindProviderT time.Time //the time of FindProvider

	ResolveCommucationT time.Duration //time used in communication within critical resolve path
	ResolveScheduleT time.Duration //the rest time used in resolving provider

	FoundProviderT time.Time //the time Found the provider
	ConnectT time.Time //the time successfully build connection

	FinishT time.Time //the time finish exchanging blocks with bitswap



	Resolver *ResolveTracker
}

//NOTE: don't work in parallel requesting
type Tracker struct {

	//Trackers map[cid.Cid]*FTracker

	Trackers *sync.Map

	totalRedundants int
	totalVariance float64
}

func (tr *Tracker) WantBlocks(c cid.Cid,t time.Time){
	_,ok:=tr.Trackers.Load(c)
	if ok{
		return
	}
	ft:=&FTracker{WantBlocksT: t}
	resolve:=new(ResolveTracker)
	resolve.Init(c)
	ft.Resolver=resolve

	tr.Trackers.Store(c,ft)
}

func (tr *Tracker) UpdateRedundant(){
	tr.totalRedundants++
}

func (tr *Tracker) UpdateVariance(vi float64){
	tr.totalVariance+=vi
}

func (tr *Tracker) FindProvider(c cid.Cid, t time.Time){
	ft,ok:=tr.Trackers.Load(c)
	if !ok{
		return
	}

	if ft.(*FTracker).FindProviderT.IsZero(){
		ft.(*FTracker).FindProviderT=t
	}
}

func(tr *Tracker) GetResolver(c cid.Cid) *ResolveTracker{
	ft,ok:=tr.Trackers.Load(c)
	if ok{
		return ft.(*FTracker).Resolver
	}
	return nil
}

func(tr *Tracker) GetResolverMH(mh multihash.Multihash) (*ResolveTracker,error){
	var result *ResolveTracker
	tr.Trackers.Range(func(key, value interface{}) bool {
		h:=key.(cid.Cid).Hash()
		if mh.String()==h.String(){
			result=value.(*FTracker).Resolver
			return false
		}
		return true
	})
	if result==nil{
		return nil,errors.New("NOTFOUND")
	}else{
		return result,nil
	}
}

func(tr *Tracker)FoundProvider(c cid.Cid, t time.Time){
	ft,ok:=tr.Trackers.Load(c)
	if !ok{
		return
	}
	ft.(*FTracker).FoundProviderT=t
	tr.Trackers.Store(c,ft)
}


func(tr *Tracker)Connected(c cid.Cid,t time.Time){
	ft,ok:=tr.Trackers.Load(c)
	if !ok{
		return
	}
	ft.(*FTracker).ConnectT=t
	tr.Trackers.Store(c,ft)
}



//TODO
func(tr *Tracker)Finish(c string,t time.Time){
	Found:=false
	tr.Trackers.Range(func(key, value interface{}) bool {
		if key.(cid.Cid).String()==c{
			value.(*FTracker).FinishT=t
			tr.Trackers.Store(key,value)
			Found=true
			return false
		}
		return true
	})
	if !Found{
		fmt.Println("Finish for no one")
	}
}
var MyTracker=NewTracker()


func NewTracker() *Tracker{
	t:=&Tracker{
		Trackers: new(sync.Map),
		totalRedundants: 0,
	}
	return t
}

func(tr *Tracker) PrintAll(){
	tr.Trackers.Range(func(key, value interface{}) bool {
		fmt.Println(key.(cid.Cid))
		fmt.Printf("	%f\n",value.(*FTracker).FindProviderT.Sub(value.(*FTracker).WantBlocksT).Seconds()*1000)
		//v.Resolver.State()
		CO,SO:=value.(*FTracker).Resolver.Collect()
		fmt.Printf("	%f\n ",CO.Seconds()*1000)
		fmt.Printf("	%f\n",SO.Seconds()*1000)
		fmt.Printf("	%f\n",value.(*FTracker).ConnectT.Sub(value.(*FTracker).FoundProviderT).Seconds()*1000)
		fmt.Printf("	%f\n",value.(*FTracker).FinishT.Sub(value.(*FTracker).ConnectT).Seconds()*1000)

		return true
	})

}

var Logger=logging.Logger("resolve")

func (tr *Tracker)CollectRedundant(){
	totalWants:=0
	tr.Trackers.Range(func(key, value interface{}) bool {
		totalWants++
		return true
	})
	fmt.Printf("average redundants: %f\n",float64(tr.totalRedundants)/float64(totalWants))
}

func (tr *Tracker)CollectVariance(){
	totalWants:=0
	tr.Trackers.Range(func(key, value interface{}) bool {
		totalWants++
		return true
	})
	fmt.Printf("average variance: %f\n",tr.totalVariance/float64(totalWants))
}


func(tr *Tracker) Collect(){
	totalt:=make([]time.Duration,5)
	effective:=0
	size:=0

	tr.Trackers.Range(func(key, value interface{}) bool {
		v:=value.(*FTracker)
		size++
		CO,SO:=v.Resolver.Collect()

		if v.FindProviderT.Sub(v.WantBlocksT)>0 && v.ConnectT.Sub(v.FoundProviderT)>0 && v.FinishT.Sub(v.ConnectT)>0 {
			if CO != 0 && SO != 0 {
				effective++
				totalt[0] += v.FindProviderT.Sub(v.WantBlocksT)
				totalt[1] += CO
				totalt[2] += SO
				totalt[3] += v.ConnectT.Sub(v.FoundProviderT)
				totalt[4] += v.FinishT.Sub(v.ConnectT)
			}
		}
		return true
	})

	fmt.Printf("Average Lantency, effective %d\n",effective)
	for i:=0;i<5;i++{
		fmt.Printf("%f\n",totalt[i].Seconds()*1000/float64(size))
	}
}