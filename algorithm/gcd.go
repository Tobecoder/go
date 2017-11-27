package algorithm

func Gcd(a, b int) int {
	for {
		if b <= 0 {
			break
		}
		a = a % b
		a, b = b, a
	}
	return a
}
