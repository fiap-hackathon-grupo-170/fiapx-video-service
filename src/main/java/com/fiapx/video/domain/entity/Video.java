package com.fiapx.video.domain.entity;

import jakarta.persistence.*;
import lombok.*;
import org.hibernate.annotations.CreationTimestamp;
import org.hibernate.annotations.UpdateTimestamp;

import java.math.BigDecimal;
import java.time.LocalDateTime;
import java.util.UUID;

@Entity
@Table(name = "videos")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Video {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @Column(name = "user_id", nullable = false)
    private UUID userId;

    @Column(nullable = false, length = 500)
    private String filename;

    @Column(name = "s3_path_original", nullable = false, length = 1000)
    private String s3PathOriginal;

    @Column(name = "zip_s3_path", length = 1000)
    private String zipS3Path;

    @Column(name = "size_bytes", nullable = false)
    private Long sizeBytes;

    @Column(name = "mime_type", nullable = false, length = 100)
    private String mimeType;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 50)
    private VideoStatus status;

    @Column(name = "frames_count")
    private Integer framesCount;

    @Column(name = "error_message", columnDefinition = "TEXT")
    private String errorMessage;

    @CreationTimestamp
    @Column(name = "uploaded_at", nullable = false, updatable = false)
    private LocalDateTime uploadedAt;

    @Column(name = "processed_at")
    private LocalDateTime processedAt;

    @UpdateTimestamp
    @Column(name = "updated_at")
    private LocalDateTime updatedAt;

    // ========== REGRAS DE NEGÓCIO (Domain Logic) ==========

    public void markAsProcessing() {
        if (this.status != VideoStatus.UPLOADED) {
            throw new IllegalStateException(
                "Apenas vídeos com status UPLOADED podem ser processados. Status atual: " + this.status
            );
        }
        this.status = VideoStatus.PROCESSING;
    }

    public void markAsCompleted(String zipS3Path, Integer framesCount) {
        if (this.status != VideoStatus.PROCESSING) {
            throw new IllegalStateException(
                "Apenas vídeos em PROCESSING podem ser marcados como COMPLETED. Status atual: " + this.status
            );
        }
        this.status = VideoStatus.COMPLETED;
        this.zipS3Path = zipS3Path;
        this.framesCount = framesCount;
        this.processedAt = LocalDateTime.now();
        this.errorMessage = null;
    }

    public void markAsFailed(String errorMessage) {
        this.status = VideoStatus.FAILED;
        this.errorMessage = errorMessage;
        this.processedAt = LocalDateTime.now();
    }

    public boolean isCompleted() {
        return this.status == VideoStatus.COMPLETED;
    }

    public boolean canBeDeleted() {
        // Não permitir deletar vídeos em processamento
        return this.status != VideoStatus.PROCESSING;
    }

    public String getFormattedSize() {
        if (sizeBytes == null) return "0 B";
        
        long kb = 1024;
        long mb = kb * 1024;
        long gb = mb * 1024;
        
        if (sizeBytes >= gb) {
            return String.format("%.2f GB", (double) sizeBytes / gb);
        } else if (sizeBytes >= mb) {
            return String.format("%.2f MB", (double) sizeBytes / mb);
        } else if (sizeBytes >= kb) {
            return String.format("%.2f KB", (double) sizeBytes / kb);
        } else {
            return sizeBytes + " B";
        }
    }
}
