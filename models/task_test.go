package models

import (
	"strings"
	"time"

	"github.com/gobuffalo/nulls"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) Test_Tasks_LoadOrdered_ByPosition() {
	for _, t := range []*Task{{Title: "tercera", Position: 3}, {Title: "primera", Position: 1}, {Title: "segunda", Position: 2}} {
		ms.NoError(ms.DB.Create(t))
	}

	tasks := Tasks{}
	ms.NoError(tasks.LoadOrdered(ms.DB))
	ms.Len(tasks, 3)
	ms.Equal("primera", tasks[0].Title)
	ms.Equal("segunda", tasks[1].Title)
	ms.Equal("tercera", tasks[2].Title)
}

func (ms *ModelSuite) Test_Tasks_LoadOrdered_TieBreaksByCreatedAtThenID() {
	now := time.Now()
	older := &Task{Title: "antigua", Position: 1, CreatedAt: now.Add(-time.Hour)}
	newer := &Task{Title: "reciente", Position: 1, CreatedAt: now}
	ms.NoError(ms.DB.Create(newer))
	ms.NoError(ms.DB.Create(older))

	tasks := Tasks{}
	ms.NoError(tasks.LoadOrdered(ms.DB))
	ms.Len(tasks, 2)
	ms.Equal("antigua", tasks[0].Title)
	ms.Equal("reciente", tasks[1].Title)

	// Con position y created_at iguales, decide el id.
	ms.NoError(ms.DB.RawQuery("UPDATE tasks SET created_at = ?", now).Exec())
	tasks = Tasks{}
	ms.NoError(tasks.LoadOrdered(ms.DB))
	ms.True(tasks[0].ID.String() < tasks[1].ID.String())
}

func (ms *ModelSuite) Test_Task_Create_WithDueDate() {
	due := time.Date(2099, 6, 15, 0, 0, 0, 0, time.UTC)
	task := &Task{Title: "Con fecha", DueDate: nulls.NewTime(due)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.DueDate.Valid)
	ms.Equal(due.Format("2006-01-02"), reloaded.DueDate.Time.UTC().Format("2006-01-02"))
}

func (ms *ModelSuite) Test_Task_Create_WithoutDueDate() {
	task := &Task{Title: "Sin fecha"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.False(reloaded.DueDate.Valid)
}

func (ms *ModelSuite) Test_Task_IsOverdue() {
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC)

	overdue := &Task{Title: "Vencida", DueDate: nulls.NewTime(past), Completed: false}
	ms.True(overdue.IsOverdue())

	notYet := &Task{Title: "Futura", DueDate: nulls.NewTime(future), Completed: false}
	ms.False(notYet.IsOverdue())

	completedPast := &Task{Title: "Completada pasada", DueDate: nulls.NewTime(past), Completed: true}
	ms.False(completedPast.IsOverdue())

	noDue := &Task{Title: "Sin fecha", Completed: false}
	ms.False(noDue.IsOverdue())
}

func (ms *ModelSuite) Test_NextPosition() {
	next, err := NextPosition(ms.DB)
	ms.NoError(err)
	ms.Equal(1, next)

	ms.NoError(ms.DB.Create(&Task{Title: "a", Position: 4}))
	ms.NoError(ms.DB.Create(&Task{Title: "b", Position: 2}))

	next, err = NextPosition(ms.DB)
	ms.NoError(err)
	ms.Equal(5, next)
}
