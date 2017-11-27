package algorithm

//获取最大子序列和
func Mss(s []int) int {
	maxSum, thisSum := 0, 0
	for _, v := range s {
		thisSum += v
		if thisSum > maxSum {
			maxSum = thisSum
		} else if thisSum <= 0 {
			thisSum = 0
		}
	}
	return maxSum
}
