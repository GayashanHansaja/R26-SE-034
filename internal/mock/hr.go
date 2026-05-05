package mock

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Employee struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Role       string `json:"role"`
	Department string `json:"department"`
	Email      string `json:"email"`
}

var employees = []Employee{
	{ID: "emp-001", Name: "Nimendra Anuradha", Role: "Software Engineer", Department: "Engineering", Email: "nimendra@example.com"},
	{ID: "emp-002", Name: "Jane Doe", Role: "HR Manager", Department: "HR", Email: "jane@example.com"},
}

func EmployeesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"data":  employees,
		"total": len(employees),
	})
}

func EmployeeByIDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/hr/employees/")
	for _, emp := range employees {
		if emp.ID == id {
			json.NewEncoder(w).Encode(emp)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{"error": "not found", "id": id})
}

func LeaveRequestsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[], "total":0}`))
}

func DepartmentsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"data":[{"id":"dept-001","name":"Engineering"},{"id":"dept-002","name":"HR"}], "total":2}`))
}
