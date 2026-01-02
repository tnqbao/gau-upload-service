# Consumer Refactor - Loại bỏ Disk I/O

## 📋 Tóm tắt

Consumer đã được refactor để **stream trực tiếp** từ temp MinIO sang main MinIO **KHÔNG QUA DISK**, phù hợp với luồng chunked upload mới (backend đã merge chunks).

## ✅ Thay đổi chính

### 1. **Loại bỏ hoàn toàn Disk I/O** (`consumer/service/chunker.go`)

#### Trước đây:
```go
// ❌ Download file từ temp MinIO -> Ghi vào disk
localPath, err := s.downloadFromTemp(ctx, req.TempBucket, req.TempPath)
defer os.Remove(localPath) 

// ❌ Đọc file từ disk vào memory
file, err := os.Open(localPath)
data := make([]byte, info.Size())
io.ReadFull(file, data)

// ❌ Upload từ memory lên main MinIO
s.infra.MinioClient.PutObjectWithMetadata(ctx, bucket, key, data, ...)
```

#### Hiện tại:
```go
// ✅ Stream trực tiếp temp MinIO -> main MinIO (KHÔNG QUA DISK)
stream, size, err := s.infra.TempMinioClient.GetObjectStream(ctx, tempBucket, tempPath)
defer stream.Close()

// ✅ Sử dụng S3 Upload Manager để xử lý file lớn (multipart upload tự động)
s.infra.MinioClient.PutObjectStreamWithMetadata(ctx, mainBucket, mainKey, stream, size, ...)
```

### 2. **Sử dụng S3 Upload Manager** (`shared/infra/minio.go`)

Đối với file lớn (>5MB), Upload Manager tự động chia thành nhiều parts và upload song song:

```go
uploader := manager.NewUploader(m.Client, func(u *manager.Uploader) {
    u.PartSize = 10 * 1024 * 1024 // 10MB per part
    u.Concurrency = 3              // Upload 3 parts đồng thời
})
```

**Lợi ích:**
- Xử lý file cực lớn (>5GB) không gặp vấn đề
- Retry tự động từng part nếu fail
- Upload song song tăng tốc độ

### 3. **Fix Retry Logic** (`consumer/main.go`)

Trước đây khi lỗi, message được requeue vô hạn gây loop:

```go
// ❌ CŨ: Infinite retry loop
msg.Nack(false, true) // requeue = true

// ✅ MỚI: Không requeue, chuyển sang dead-letter queue
msg.Nack(false, false) // requeue = false
```

### 4. **Loại bỏ code không cần thiết**

- ❌ Xóa `downloadFromTemp()` method
- ❌ Xóa `uploadToMain()` method (old version)
- ❌ Xóa `tempDir` field từ `ChunkerService`
- ❌ Xóa imports: `os`
- ✅ Thêm `streamToMain()` method (stream trực tiếp)
- ✅ Thêm S3 Upload Manager dependency

### 3. **Giảm Resource Requirements** (`deploy/k8s/staging/template/deployment.yaml`)

```yaml
# Consumer không còn cần nhiều CPU/Memory vì không có disk I/O
resources:
  requests:
    cpu: "100m"      # Giảm từ 250m
    memory: "128Mi"  # Giảm từ 256Mi
  limits:
    cpu: "500m"      # Giảm từ 750m
    memory: "256Mi"  # Giảm từ 512Mi
```

## 🔄 Luồng hoạt động mới

```
┌────────────┐          ┌─────────────┐          ┌────────────┐
│ Temp MinIO │──stream──►│  Consumer   │──stream──►│ Main MinIO │
│  (merged)  │          │ (no disk I/O)│          │   (final)  │
└────────────┘          └─────────────┘          └────────────┘
                              │
                              ▼
                        ┌──────────┐
                        │  Delete  │
                        │temp file │
                        └──────────┘
```

## 📊 Lợi ích

1. **Hiệu suất cao hơn**
   - Không có disk I/O overhead
   - Stream trực tiếp giữa 2 MinIO
   - Xử lý file lớn (> 5GB) mà không cần lo memory

2. **Tiết kiệm tài nguyên**
   - Không cần disk space cho temp files
   - Giảm CPU/Memory requirements
   - Giảm chi phí infrastructure

3. **Đơn giản hơn**
   - Code ngắn gọn, dễ maintain
   - Ít error handling
   - Không cần cleanup disk files

4. **Phù hợp với luồng mới**
   - Backend đã merge chunks
   - Consumer chỉ cần move file
   - Không chia chunk nữa

## 🔍 Flow chi tiết

### Message từ RabbitMQ:
```json
{
  "temp_bucket": "temp-uploads",
  "temp_path": "sha256hash.mp4",
  "target_bucket": "user-bucket-name",
  "target_folder": "videos/2026/sha256hash",
  "original_name": "large-video.mp4",
  "file_hash": "sha256...",
  "file_size": 524288000,
  "chunk_size": 0,
  "metadata": { ... }
}
```

### Consumer xử lý:
1. **Parse message** từ RabbitMQ
2. **Determine path**: `customPath/hash.ext`
3. **Stream file**: temp MinIO → main MinIO (direct)
4. **Cleanup**: Delete temp file
5. **Ack message**: Mark as processed

## ⚠️ Lưu ý

- Consumer **KHÔNG** chia chunk nữa (backend đã làm rồi)
- File ở temp MinIO đã là file **hoàn chỉnh** (đã merge)
- Consumer chỉ **move** file từ temp → main bucket
- Stream sử dụng `io.Reader` nên không load toàn bộ vào memory

## 🚀 Deploy

Sau khi refactor, cần redeploy consumer:

```bash
# Staging
cd deploy/k8s/staging
./apply.sh

# Production (nếu cần)
cd deploy/k8s/production
./apply.sh
```

