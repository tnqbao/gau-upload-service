# Upload Service by Gau

## Introduction | Giới thiệu

**English:**  
This repository provides an upload service written in Go, designed to handle file uploads including images and documents. It supports uploading to cloud storage (Cloudflare R2) with proper file validation and organization. The service is suitable for microservices architectures and can be deployed using Docker or Kubernetes.

**Tiếng Việt:**  
Repo này cung cấp dịch vụ upload file viết bằng Go, dùng để xử lý upload file bao gồm hình ảnh và tài liệu. Hỗ trợ upload lên cloud storage (Cloudflare R2) với kiểm tra file hợp lệ và tổ chức file. Phù hợp với kiến trúc microservices và có thể triển khai bằng Docker hoặc Kubernetes.

---

## Directory Structure | Cấu trúc thư mục

```
gau-upload-service/
├── Dockerfile
├── entrypoint.sh
├── go.mod
├── go.sum
├── main.go
├── README.md
├── config/
│   ├── env_config.go
│   └── main.go
├── controller/
│   ├── image.go
│   └── main.go
├── deploy/
│   └── k8s/
│       ├── production/
│       │   ├── apply.sh
│       │   ├── apply_envsubst.sh
│       │   ├── kustomization.yaml
│       │   ├── unapply.sh
│       │   ├── base/
│       │   └── template/
│       │       ├── configmap.yaml
│       │       ├── deployment.yaml
│       │       ├── hpa.yaml
│       │       ├── ingress.yaml
│       │       ├── secret.yaml
│       │       └── service.yaml
│       └── staging/
│           ├── apply.sh
│           ├── apply_envsubst.sh
│           ├── kustomization.yaml
│           ├── unapply.sh
│           ├── base/
│           └── template/
│               ├── deployment.yaml
│               ├── hpa.yaml
│               ├── ingress.yaml
│               ├── secret.yaml
│               └── service.yaml
├── infra/
│   ├── cloudflare_r2.go
│   ├── main.go
│   ├── postgres.go
│   └── redis.go
├── middlewares/
│   ├── main.go
│   └── private.go
├── migrations/
├── repository/
│   └── main.go
├── routes/
│   └── routes.go
└── utils/
    ├── fileCheck.go
    └── http_response.go
```

### 📑 Directory Description | Mô tả thư mục

| Path                          | Description                                             | Mô tả                                  |
|-------------------------------|---------------------------------------------------------|----------------------------------------|
| `Dockerfile`, `entrypoint.sh` | Docker image build and startup script                   | File build và khởi động Docker         |
| `go.mod`, `go.sum`            | Go module definitions                                   | Định nghĩa module Go                   |
| `config/`                     | Environment loading and configuration logic             | Logic cấu hình và load môi trường      |
| `controller/`                 | HTTP handlers for file upload operations                | Xử lý HTTP cho upload file             |
| `deploy/k8s/`                 | Kubernetes manifests and scripts for staging/production | Manifest và script triển khai trên K8s |
| `infra/`                      | Cloud storage (R2), PostgreSQL, Redis setup            | Thiết lập cloud storage, DB và Redis   |
| `middlewares/`                | Authentication and other middleware logic               | Middleware xác thực                    |
| `migrations/`                 | SQL migration files                                     | Các file migration SQL                 |
| `repository/`                 | Data access and business logic                          | Truy cập và xử lý dữ liệu              |
| `routes/`                     | API route definitions                                   | Định nghĩa route                       |
| `utils/`                      | File validation and utility functions                   | Kiểm tra file và hàm tiện ích          |

---

## Features | Tính năng

### 📤 File Upload | Upload File

**English:**
- Support for images (JPEG, PNG, WebP)
- File size validation with configurable limits
- Automatic file name sanitization (removes special characters and spaces)
- Organized storage with custom folder paths
- Upload to Cloudflare R2 cloud storage

**Tiếng Việt:**
- Hỗ trợ hình ảnh (JPEG, PNG, WebP)
- Kiểm tra kích thước file với giới hạn có thể cấu hình
- Tự động làm sạch tên file (loại bỏ ký tự đặc biệt và khoảng trống)
- Lưu trữ có tổ chức với đường dẫn thư mục tùy chỉnh
- Upload lên Cloudflare R2 cloud storage

### 🔒 Security | Bảo mật

**English:**
- File type validation based on content type
- File size limits to prevent abuse
- Input sanitization for file names and paths

**Tiếng Việt:**
- Kiểm tra loại file dựa trên content type
- Giới hạn kích thước file để tránh lạm dụng
- Làm sạch đầu vào cho tên file và đường dẫn

---

## API Endpoints | Điểm cuối API

### POST /upload/image

**Request:**
```bash
curl -X POST \
  -F "file=@image.jpg" \
  -F "file_path=user_avatars" \
  http://localhost:8080/upload/image
```

**Response:**
```json
{
  "file_path": "user_avatars/image_cleaned_name.jpg",
  "message": "File uploaded successfully"
}
```

**Parameters:**
- `file`: The image file to upload
- `file_path`: Folder name where the file will be stored (acts as bucket folder)

---

## Deployment | Triển khai

### 🐳 Docker

**English:**
1. Build the Docker image:
   ```bash
   docker build -t gau-upload-service .
   ```
2. Run the container:
   ```bash
   docker run -p 8080:8080 gau-upload-service
   ```

**Tiếng Việt:**
1. Build image Docker:
   ```bash
   docker build -t gau-upload-service .
   ```
2. Chạy container:
   ```bash
   docker run -p 8080:8080 gau-upload-service
   ```

---

### ☸ Kubernetes

**English:**
1. Edit environment variables in `deploy/k8s/staging/template/configmap.yaml` and `secret.yaml`.
2. Apply manifests:
   ```bash
   cd deploy/k8s/staging
   ./apply.sh
   ```
3. To remove:
   ```bash
   ./unapply.sh
   ```

**Tiếng Việt:**
1. Chỉnh sửa biến môi trường trong `deploy/k8s/staging/template/configmap.yaml` và `secret.yaml`.
2. Áp dụng manifest:
   ```bash
   cd deploy/k8s/staging
   ./apply.sh
   ```
3. Để xóa:
   ```bash
   ./unapply.sh
   ```

---

## Configuration | Cấu hình

### Environment Variables | Biến môi trường

| Variable | Description | Default |
|----------|-------------|---------|
| `IMAGE_MAX_SIZE` | Maximum image size in MB | 10 |
| `R2_ENDPOINT` | Cloudflare R2 endpoint | - |
| `R2_ACCESS_KEY` | R2 access key | - |
| `R2_SECRET_KEY` | R2 secret key | - |
| `R2_BUCKET_NAME` | R2 bucket name | - |

---

## Contact | Liên hệ

Nếu bạn có bất kỳ câu hỏi hoặc đề xuất nào, vui lòng liên hệ qua:

* Github: [tnqbao](https://github.com/tnqbao)
* LinkedIn: [https://www.linkedin.com/in/tnqb2004/](https://www.linkedin.com/in/tnqb2004/)
