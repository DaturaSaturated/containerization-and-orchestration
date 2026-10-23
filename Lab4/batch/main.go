package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func burnCPU(intensity int) {
	iterations := intensity * 10000
	result := 0
	for i := 0; i < iterations; i++ {
		result += i * i
	}
}

func allocateMemory(memoryMB int) []byte {
	data := make([]byte, 0)
	chunkSize := 10 * 1024 * 1024 // 10MB
	
	fmt.Printf("Batch starting: MEMORY_MB=%d\n", memoryMB)
	
	for len(data) < memoryMB*1024*1024 {
		data = append(data, make([]byte, chunkSize)...)
		fmt.Printf("Allocated %dMB\n", len(data)/(1024*1024))
		time.Sleep(500 * time.Millisecond)
	}
	
	return data
}

func main() {
	cpuIntensity := 100
	memoryMB := 100
	
	if val := os.Getenv("CPU_INTENSITY"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			cpuIntensity = v
		}
	}
	
	if val := os.Getenv("MEMORY_MB"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			memoryMB = v
		}
	}
	
	memory := allocateMemory(memoryMB)
	
	// Бесконечный цикл: жжём CPU и держим память
	for {
		burnCPU(cpuIntensity)
		time.Sleep(100 * time.Millisecond)
		_ = memory // чтобы не оптимизировался
	}
}