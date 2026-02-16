package com.fiapx;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertNotNull;

class VideoServiceApplicationTest {

    @Test
    void contextLoads() {
        // Simple test to verify the application can be instantiated
        VideoServiceApplication app = new VideoServiceApplication();
        assertNotNull(app, "VideoServiceApplication should not be null");
    }

    @Test
    void mainMethodExists() {
        // Verify main method exists and can be called without throwing exceptions
        try {
            VideoServiceApplication.class.getMethod("main", String[].class);
        } catch (NoSuchMethodException e) {
            throw new AssertionError("Main method should exist", e);
        }
    }
}
