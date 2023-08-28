package main

import (
	"context"
	"fmt"
	"time"
)

// ЗАДАНИЕ:
// * сделать из плохого кода хороший;
// * важно сохранить логику появления ошибочных тасков;
// * сделать правильную мультипоточность обработки заданий.
// Обновленный код отправить через merge-request.

// приложение эмулирует получение и обработку тасков, пытается и получать и обрабатывать в многопоточном режиме
// В конце должно выводить успешные таски и ошибки выполнены остальных тасков

// A Task represents a meaninglessness of our life
type Task struct {
	id         int
	createTime string // время создания
	endTime    string // время выполнения
	done       bool
}

func taskCreator(tasksChan chan Task, ctx context.Context) {
	var taskId int
loop:
	for {
		taskTime := time.Now().Format(time.RFC3339)
		if time.Now().Nanosecond()%2 > 0 { // вот такое условие появления ошибочных тасков
			taskTime = "Some error occurred"
		}
		select {
		case <-ctx.Done():
			break loop
		case tasksChan <- Task{createTime: taskTime, id: taskId}: // передаем таск на выполнение
			taskId++
		default:
		}
	}
}

func taskWorker(task Task) Task {
	parsedCreateTime, _ := time.Parse(time.RFC3339, task.createTime)
	delta := time.Now().Unix() - parsedCreateTime.Unix()

	if delta < 20 {
		task.done = true
	} else {
		task.done = false
	}
	task.endTime = time.Now().Format(time.RFC3339Nano)

	time.Sleep(time.Millisecond * 150)

	return task
}

func taskSorter(task Task, doneTasks chan Task, failedTasks chan error) {
	if task.done {
		doneTasks <- task
	} else {
		failedTasks <- fmt.Errorf("task id %d, time %s, done %t", task.id, task.createTime, task.done)
	}
}

func main() {
	tasksChan := make(chan Task, 10)
	doneTasks := make(chan Task)
	failedTasks := make(chan error)
	ctx := context.Background()
	// write to channel "tasksChan" and do work for 1 seconds and then handle the rest of the values from the channel
	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()

	go taskCreator(tasksChan, ctx)

	go func(tasksChan, doneTasks chan Task, failedTasks chan error) {
		// получение тасков
		for task := range tasksChan {
			task = taskWorker(task)
			taskSorter(task, doneTasks, failedTasks)
		}
		close(tasksChan)
	}(tasksChan, doneTasks, failedTasks)

	result := map[int]Task{}
	go func(doneTasks chan Task, result *map[int]Task) {
		for r := range doneTasks {
			(*result)[r.id] = r
		}
		close(doneTasks)
	}(doneTasks, &result)

	var errors []error
	go func(failedTasks chan error, errors *[]error) {
		for r := range failedTasks {
			*errors = append(*errors, r)
		}
		close(failedTasks)
	}(failedTasks, &errors)

	time.Sleep(time.Second * 3)

	fmt.Println("Errors:")
	for _, err := range errors {
		fmt.Println(err)
	}

	fmt.Println("Done tasks:")
	for resultId := range result {
		fmt.Printf("task id %d\n", resultId)
	}
}
