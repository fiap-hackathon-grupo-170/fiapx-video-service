package com.fiapx;

import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertTrue;

/**
 * Simple test for CI debugging purposes.
 * This test verifies basic application structure without loading the full Spring context.
 */
class ApplicationContextTest {

    @Test
    void applicationMainClassExists() {
        // Verify that the main application class exists
        try {
            Class.forName("com.fiapx.VideoServiceApplication");
            assertTrue(true, "VideoServiceApplication class should exist");
        } catch (ClassNotFoundException e) {
            throw new AssertionError("VideoServiceApplication class not found", e);
        }
    }

    @Test
    void applicationHasSpringBootAnnotation() {
        // Verify that the main application class has @SpringBootApplication annotation
        try {
            Class<?> appClass = Class.forName("com.fiapx.VideoServiceApplication");
            boolean hasAnnotation = appClass.isAnnotationPresent(
                org.springframework.boot.autoconfigure.SpringBootApplication.class
            );
            assertTrue(hasAnnotation, "VideoServiceApplication should have @SpringBootApplication annotation");
        } catch (ClassNotFoundException e) {
            throw new AssertionError("VideoServiceApplication class not found", e);
        }
    }
}
