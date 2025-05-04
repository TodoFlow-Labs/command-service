// internal/dto/todo.go
package dto

// type CreateTodo struct {
// 	Title string `json:"title"`
// }

// type UpdateTodo struct {
// 	ID        string `json:"id"`
// 	Title     string `json:"title,omitempty"`
// 	Completed *bool  `json:"completed,omitempty"`
// }

// type DeleteTodo struct {
// 	ID string `json:"id"`
// }

type Command struct {
	Type      string `json:"type"`
}

type Todo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type TodoList struct {
	Todos []Todo `json:"todos"`
}
