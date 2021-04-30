package quieoo

import (
	"time"
)

const(
	L0=1
	CollectWindow=3
	HitRatioThreshold=0.75
	INC0=2
	INC1=1
	MaxWaitTime=1*time.Second

	AllowedDelayVariation=0.05
)
type DynamicAdjuster struct {
	role	ProviderRole
	minRequestTime float64

	historyRequestTime []float64
	historyDifference []float64
	continuousHitTimes int

	L int
}

func NewDynamicAdjuster() *DynamicAdjuster{
	return &DynamicAdjuster{
		role: Role_CoWorker,
		minRequestTime: 1000*time.Second.Seconds(),
		historyRequestTime: make([]float64,CollectWindow),
		continuousHitTimes: 0,
		historyDifference: make([]float64,CollectWindow),
	}
}

func (da *DynamicAdjuster) AverageRequestTime()float64{
	r:=0.0
	for _,c:=range da.historyRequestTime{
		r+=c
	}
	return r/CollectWindow
}

func (da *DynamicAdjuster) AverageDiff()float64{
	r:=0.0
	for _,c:=range da.historyDifference{
		r+=c
	}
	return r/CollectWindow
}

func (da *DynamicAdjuster) Adjust(hr float64,d time.Duration, n int) int{
	LastRequestTime:=d.Seconds()/(float64(n))

	average:=da.AverageRequestTime()

	if da.role==Role_CoWorker{
		if hr>HitRatioThreshold{
			da.L+=INC0
		}
	}else{

		if LastRequestTime>average && ((LastRequestTime-average)/(LastRequestTime-da.minRequestTime) > AllowedDelayVariation){
			if da.L>1{
				da.L=da.L/2
			}
		}
		if LastRequestTime<average && ((average-LastRequestTime)/(LastRequestTime-da.minRequestTime) < AllowedDelayVariation){
			da.L+=INC1
		}
	}


	// update sates
	if hr>HitRatioThreshold{
		da.continuousHitTimes++
		if da.continuousHitTimes>CollectWindow{
			da.role=Role_FullProvider
		}
	}else{
		da.role=Role_CoWorker
	}

	for i:=0;i<CollectWindow-1;i++{
		da.historyRequestTime[i]=da.historyRequestTime[i+1]
	}
	da.historyRequestTime[CollectWindow-1]=LastRequestTime
	if LastRequestTime<da.minRequestTime{
		da.minRequestTime=LastRequestTime
	}

	return da.L
}

func (da *DynamicAdjuster) Adjust3(hr float64,d time.Duration, n int) int{
	LastRequestTime:=d.Seconds()/(float64(n))

	averagediff:=da.AverageDiff()


	if da.role==Role_CoWorker{
		if hr>HitRatioThreshold{
			da.L+=INC0
		}
	}else{
		if averagediff/da.minRequestTime<0{
			da.L+=INC1
		}else{
			da.L=da.L*int(1-0.8*averagediff/da.minRequestTime)
		}
	}


	// update sates
	if hr>HitRatioThreshold{
		da.continuousHitTimes++
		if da.continuousHitTimes>CollectWindow{
			da.role=Role_FullProvider
		}
	}else{
		da.role=Role_CoWorker
	}

	for i:=0;i<CollectWindow-1;i++{
		da.historyRequestTime[i]=da.historyRequestTime[i+1]
		da.historyDifference[i]=da.historyDifference[i+1]
	}
	da.historyDifference[CollectWindow-1]=LastRequestTime-da.historyRequestTime[CollectWindow-1]
	da.historyRequestTime[CollectWindow-1]=LastRequestTime
	if LastRequestTime<da.minRequestTime{
		da.minRequestTime=LastRequestTime
	}

	return da.L
}

func (da *DynamicAdjuster) Adjust2(hr float64,d time.Duration, n int) int{
	if n==0{
		return da.L
	}
	LastRequestTime:=d.Seconds()/float64(n)
	average:=da.AverageRequestTime()

	if da.role==Role_CoWorker{
		if hr>HitRatioThreshold{
			da.L+=INC0
		}
	}else{
		if (LastRequestTime-average)/(LastRequestTime-da.minRequestTime) > AllowedDelayVariation{
			if da.L>1{
				da.L=da.L/2
			}
		}
		da.L+=INC1

	}


	// update sates
	if hr>HitRatioThreshold{
		da.continuousHitTimes++
		if da.continuousHitTimes>CollectWindow{
			da.role=Role_FullProvider
		}
	}else{
		da.role=Role_CoWorker
	}

	for i:=0;i<CollectWindow-1;i++{
		da.historyRequestTime[i]=da.historyRequestTime[i+1]
	}
	da.historyRequestTime[CollectWindow-1]=LastRequestTime
	if LastRequestTime<da.minRequestTime{
		da.minRequestTime=LastRequestTime
	}

	return da.L
}

