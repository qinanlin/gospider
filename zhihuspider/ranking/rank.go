package ranking

type rankUnit struct {
	val  int
	name string
	url  string
	//add
}

type RankData []rankUnit

func (data RankData) Len() int {
	return len(data)
}

func (data RankData) Less(i, j int) bool { //小顶堆
	return data[i].val < data[j].val
}

func (data RankData) Swap(i, j int) {
	data[i], data[j] = data[j], data[i]
}

func (data *RankData) Push(x interface{}) {
	*data = append(*data, x.(rankUnit))
}

func (data *RankData) Pop() interface{} {
	old := *data
	l := len(old)
	x := old[l-1]
	*data = old[0 : l-1]
	return x
}
