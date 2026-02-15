package com.fiapx.video.domain.entity;

public enum VideoStatus {
    UPLOADED,      // Vídeo foi enviado
    PROCESSING,    // Está sendo processado
    COMPLETED,     // Processamento concluído com sucesso
    FAILED         // Falha no processamento
}
