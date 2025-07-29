# Gau Upload Service

## Introduction | Giới thiệu

**English:**  
This repository provides a file upload service written in Go, designed to handle uploading images, documents, and other files. It is suitable for microservices architectures and can be deployed using Docker or Kubernetes.

**Tiếng Việt:**  
Repo này cung cấp dịch vụ upload file (hình ảnh, tài liệu, ...) viết bằng Go. Phù hợp với kiến trúc microservices và có thể triển khai bằng Docker hoặc Kubernetes.

---

## Directory Structure | Cấu trúc thư mục

```
gau-upload-service/
├── Dockerfile
├── entrypoint.sh
├── main.go
├── go.mod
├── go.sum
├── config/
├── controller/
├── infra/
├── migrations/
├── repository/
├── routes/
├── utils/
```

### 📑 Directory Description | Mô tả thư mục

| Path                          | Description                                             | Mô tả                                  |
|-------------------------------|---------------------------------------------------------|----------------------------------------|
| `Dockerfile`, `entrypoint.sh` | Docker image build and startup script                   | File build và khởi động Docker         |
| `go.mod`, `go.sum`            | Go module definitions                                   | Định nghĩa module Go                   |
| `config/`                     | Environment loading and configuration logic             | Logic cấu hình và load môi trường      |
| `controller/`                 | HTTP handlers for file upload                           | Xử lý HTTP upload file                 |
| `infra/`                      | Cloudflare R2, PostgreSQL, Redis setup and connections  | Thiết lập Cloudflare R2, DB, Redis     |
| `migrations/`                 | SQL migration files                                     | Các file migration SQL                 |
| `repository/`                 | Data access and business logic                          | Truy cập và xử lý dữ liệu              |
| `routes/`                     | API route definitions                                   | Định nghĩa route                       |
| `utils/`                      | Utility functions (file check, HTTP response, etc.)     | Hàm tiện ích                           |

---

## Deployment | Triển khai

### 🧪 Init Environment | Khởi tạo môi trường

> Ensure you have set up your environment variables as needed for your deployment.

---

### 🐳 Docker

**English:**
1. Build the Docker image:
   ```bash
   docker build -t gau-upload-service .
   ```
2. Run the container:
   ```bash
   docker run -d --env-file .env -p 8080:8080 gau-upload-service
   ```

**Tiếng Việt:**
1. Build image Docker:
   ```bash
   docker build -t gau-upload-service .
   ```
2. Chạy container:
   ```bash
   docker run -d --env-file .env -p 8080:8080 gau-upload-service
   ```

---

### ☸ Kubernetes

**English:**
1. Edit environment variables in your Kubernetes manifests as needed.
2. Apply manifests:
   ```bash
   kubectl apply -f k8s/
   ```
3. To remove:
   ```bash
   kubectl delete -f k8s/
   ```

**Tiếng Việt:**
1. Chỉnh sửa biến môi trường trong manifest Kubernetes nếu cần.
2. Áp dụng manifest:
   ```bash
   kubectl apply -f k8s/
   ```
3. Để xóa:
   ```bash
   kubectl delete -f k8s/
   ```

---

## Liên hệ | Contact

Nếu bạn có bất kỳ câu hỏi hoặc đề xuất nào, vui lòng liên hệ qua email:

* Github: [tnqbao](https://github.com/tnqbao)
* LinkedIn: [https://www.linkedin.com/in/tnqb2004/](https://www.linkedin.com/in/tnqb2004/)

