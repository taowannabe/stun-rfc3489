package main

import "fmt"

func main() {
	iArray := [2]int{1,2}
	sArray := iArray[:1]
	aArray := append(sArray,4,5)
	fmt.Printf("%p,%p,%p,",&iArray,sArray,aArray)
	fmt.Printf("%v",iArray)
	iArray[1] = 3
	fmt.Printf("%v",iArray)
	//testPoint(sArray)
}

