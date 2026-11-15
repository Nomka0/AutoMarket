#!/usr/bin/env python3
"""
Benchmark Criptográfico Forense: AutoMarket SHA-256 vs Argon2id
Tarea T4-04 - PoC Explotación & Análisis Forense

Compara empíricamente el rendimiento de cálculo y la resistencia criptográfica
entre el esquema actual de AutoMarket (SHA-256 con salt) y la función adaptativa
moderna recomendada por OWASP (Argon2id).
"""

import hashlib
import os
import platform
import secrets
import sys
import time

try:
    import argon2
except ImportError:
    print("Error: El paquete 'argon2-cffi' es requerido. Instálelo con: pip install argon2-cffi")
    sys.exit(1)


def automarket_sha256_hash(password: str, salt: str) -> str:
    """
    Simulación exacta de la función HashPassword() en internal/auth/crypto.go:
    sha256.Sum256([]byte(password + salt)) -> hex string (64 caracteres).
    """
    payload = (password + salt).encode("utf-8")
    return hashlib.sha256(payload).hexdigest()


def create_argon2id_hasher() -> argon2.PasswordHasher:
    """
    Instancia Argon2id con parámetros recomendados por OWASP:
    - Memoria (m): 64 MB (65,536 KiB)
    - Tiempo / Iteraciones (t): 3 iteraciones
    - Paralelismo (p): 4 hilos
    - Tipo: Argon2id (híbrido resistente a side-channel y ataques GPU/ASIC)
    """
    return argon2.PasswordHasher(
        time_cost=3,
        memory_cost=65536,  # 64 MB en KiB
        parallelism=4,
        hash_len=32,
        salt_len=16,
        type=argon2.Type.ID
    )


def benchmark_sha256(iterations: int = 10_000) -> dict:
    """
    Mide empíricamente el tiempo para calcular N hashes SHA-256 con salt.
    """
    # Generar un salt típico de 16 bytes en hex (32 chars) como en AutoMarket
    salt = secrets.token_hex(16)
    password_base = "P@ssw0rdAutoMarket2026!"

    # Calentamiento ligero de CPU
    for _ in range(100):
        _ = automarket_sha256_hash(password_base, salt)

    start_time = time.perf_counter()
    for i in range(iterations):
        # Variamos levemente la entrada para simular intentos reales de ataque de diccionario / fuerza bruta
        _ = automarket_sha256_hash(f"{password_base}_{i}", salt)
    end_time = time.perf_counter()

    elapsed = end_time - start_time
    hps = iterations / elapsed if elapsed > 0 else 0.0
    avg_us = (elapsed / iterations) * 1_000_000 if iterations > 0 else 0.0

    return {
        "algorithm": "SHA-256 (AutoMarket)",
        "parameters": "sha256(password + salt)",
        "iterations": iterations,
        "elapsed_seconds": elapsed,
        "hashes_per_second": hps,
        "avg_time_per_hash_us": avg_us
    }


def benchmark_argon2id(iterations: int = 100) -> dict:
    """
    Mide empíricamente el tiempo para calcular N hashes Argon2id con parámetros OWASP.
    """
    ph = create_argon2id_hasher()
    password_base = "P@ssw0rdAutoMarket2026!"

    # Calentamiento ligero (1 ejecución)
    _ = ph.hash("warmup_pass")

    start_time = time.perf_counter()
    for i in range(iterations):
        _ = ph.hash(f"{password_base}_{i}")
    end_time = time.perf_counter()

    elapsed = end_time - start_time
    hps = iterations / elapsed if elapsed > 0 else 0.0
    avg_ms = (elapsed / iterations) * 1_000 if iterations > 0 else 0.0

    return {
        "algorithm": "Argon2id (OWASP)",
        "parameters": "m=64MB (65536 KiB), t=3 iteraciones, p=4 hilos",
        "iterations": iterations,
        "elapsed_seconds": elapsed,
        "hashes_per_second": hps,
        "avg_time_per_hash_ms": avg_ms
    }


def project_gpu_cracking():
    """
    Proyecta la velocidad y tiempo teórico/práctico de cracking con hardware ofensivo
    (NVIDIA RTX 3080 / 4090 con Hashcat) frente a una contraseña alfanumérica de 8 caracteres.
    """
    charset_len = 62  # [a-zA-Z0-9]
    pwd_len = 8
    keyspace = charset_len ** pwd_len
    keyspace_avg = keyspace / 2

    # Tasas de rendimiento basadas en benchmarks reales de Hashcat
    # Modo 1400 (SHA-256): Gigahashes/segundo (GH/s)
    # Modo 22400 (Argon2id OWASP m=64MB, t=3, p=4): Limitado severamente por VRAM y ancho de banda
    gpu_profiles = {
        "RTX 3080 (10 GB VRAM)": {
            "sha256_rate": 5.2e9,     # 5.2 GH/s
            "argon2id_rate": 75.0,     # ~75 H/s (0.075 kH/s por contención de 64MB/hilo)
        },
        "RTX 4090 (24 GB VRAM)": {
            "sha256_rate": 9.2e9,     # 9.2 GH/s
            "argon2id_rate": 160.0,    # ~160 H/s (0.16 kH/s)
        },
        "Cluster 8x RTX 4090": {
            "sha256_rate": 73.6e9,    # 73.6 GH/s
            "argon2id_rate": 1280.0,   # ~1.28 kH/s
        }
    }

    print("\n" + "=" * 78)
    print("  PROYECCIÓN OFENSIVA DE CRACKING GPU (HASHCAT) - CWE-916")
    print("=" * 78)
    print(f"Longitud objetivo : {pwd_len} caracteres alfanuméricos [a-zA-Z0-9]")
    print(f"Espacio de claves : 62^{pwd_len} = {keyspace:,} combinaciones")
    print(f"Espacio medio (50%): {int(keyspace_avg):,} combinaciones")
    print("-" * 78)

    def format_duration(seconds: float) -> str:
        if seconds < 60:
            return f"{seconds:.1f} s"
        elif seconds < 3600:
            return f"{seconds/60:.1f} min"
        elif seconds < 86400:
            return f"{seconds/3600:.2f} h"
        elif seconds < 86400 * 365:
            return f"{seconds/86400:.1f} días"
        else:
            return f"{seconds/(86400*365):,.1f} años"

    print(f"{'Hardware Ofensivo':<22} | {'Algoritmo':<10} | {'Tasa Estimada':<15} | {'Tiempo Medio (50%)':<18} | {'Tiempo Máx (100%)':<16}")
    print("-" * 88)

    for hw, rates in gpu_profiles.items():
        # SHA-256
        t_sha_avg = keyspace_avg / rates["sha256_rate"]
        t_sha_max = keyspace / rates["sha256_rate"]
        sha_rate_str = f"{rates['sha256_rate']/1e9:.1f} GH/s"
        print(f"{hw:<22} | {'SHA-256':<10} | {sha_rate_str:<15} | {format_duration(t_sha_avg):<18} | {format_duration(t_sha_max):<16}")

        # Argon2id
        t_arg_avg = keyspace_avg / rates["argon2id_rate"]
        t_arg_max = keyspace / rates["argon2id_rate"]
        arg_rate_str = f"{rates['argon2id_rate']/1000:.3f} kH/s"
        print(f"{'':<22} | {'Argon2id':<10} | {arg_rate_str:<15} | {format_duration(t_arg_avg):<18} | {format_duration(t_arg_max):<16}")
        print("-" * 88)

    print("\n[*] VULNERABILIDAD CWE-916 (Use of Password Hash With Insufficient Computational Effort):")
    print("    1. Velocidad intrínseca descontrolada: SHA-256 carece de costo de cómputo deliberado.")
    print("       Un atacante con una sola GPU moderna supera los 5-9 GH/s.")
    print("    2. Ausencia de costo de memoria: Cada hilo SHA-256 solo requiere 64 bytes de RAM/registros,")
    print("       permitiendo saturar completamente los miles de núcleos CUDA sin cuello de botella.")
    print("    3. Neutralización del Salt: El salt únicamente protege contra Rainbow Tables precalculadas,")
    print("       pero NO penaliza el cracking por fuerza bruta o diccionario directo.")
    print("    4. Inviabilidad en Argon2id: Con 64 MB de memoria por intento, la RTX 4090 (24 GB VRAM)")
    print("       queda limitada a un máximo de ~375 hilos concurrentes, reduciendo la velocidad a H/s")
    print("       y tornando el ataque matemáticamente inviable (>20,000 años).")
    print("=" * 78)


def run_benchmark() -> dict:
    """
    Ejecuta el benchmark completo y presenta los resultados forenses.
    """
    print("=" * 78)
    print("  BENCHMARK FORENSE DE HASHING CRIPTOGRÁFICO - T4-04")
    print("  AutoMarket (SHA-256 + Salt) vs Función Robusta (Argon2id)")
    print("=" * 78)
    print(f"Sistema Operativo : {platform.system()} {platform.release()} ({platform.architecture()[0]})")
    print(f"Procesador        : {platform.processor() or 'CPU x86_64'}")
    print(f"Núcleos CPU       : {os.cpu_count()} núcleos lógicos")
    print(f"Versión Python    : {platform.python_version()}")
    print("-" * 78)

    # 1. Medición SHA-256
    print("\n[+] Ejecutando benchmark SHA-256 (10,000 iteraciones)...")
    res_sha = benchmark_sha256(iterations=10_000)
    print(f"    - Tiempo transcurrido : {res_sha['elapsed_seconds']:.4f} s")
    print(f"    - Hashes / segundo    : {res_sha['hashes_per_second']:,.2f} H/s")
    print(f"    - Tiempo medio / hash : {res_sha['avg_time_per_hash_us']:.2f} µs")

    # 2. Medición Argon2id
    print("\n[+] Ejecutando benchmark Argon2id (100 iteraciones con 64MB, t=3, p=4)...")
    res_argon = benchmark_argon2id(iterations=100)
    print(f"    - Tiempo transcurrido : {res_argon['elapsed_seconds']:.4f} s")
    print(f"    - Hashes / segundo    : {res_argon['hashes_per_second']:,.2f} H/s")
    print(f"    - Tiempo medio / hash : {res_argon['avg_time_per_hash_ms']:.2f} ms")

    # 3. Factor de aceleración / Costo relativo
    ratio = res_sha['hashes_per_second'] / res_argon['hashes_per_second'] if res_argon['hashes_per_second'] > 0 else 0

    sha_time_str = f"{res_sha['elapsed_seconds']:.4f} s"
    argon_time_str = f"{res_argon['elapsed_seconds']:.4f} s"
    sha_rate_str = f"{res_sha['hashes_per_second']:,.2f} H/s"
    argon_rate_str = f"{res_argon['hashes_per_second']:,.2f} H/s"
    sha_lat_str = f"{res_sha['avg_time_per_hash_us']:.2f} µs"
    argon_lat_str = f"{res_argon['avg_time_per_hash_ms']:.2f} ms"

    print("\n" + "=" * 78)
    print("  RESULTADOS COMPARATIVOS Y CONCLUSIÓN FORENSE")
    print("=" * 78)
    print(f"{'Métrica':<30} | {'SHA-256 (AutoMarket)':<20} | {'Argon2id (OWASP)':<20}")
    print("-" * 78)
    print(f"{'Iteraciones':<30} | {res_sha['iterations']:<20,} | {res_argon['iterations']:<20,}")
    print(f"{'Tiempo Total':<30} | {sha_time_str:<20} | {argon_time_str:<20}")
    print(f"{'Tasa de Procesamiento':<30} | {sha_rate_str:<20} | {argon_rate_str:<20}")
    print(f"{'Latencia por Hash':<30} | {sha_lat_str:<20} | {argon_lat_str:<20}")
    print("-" * 78)
    print(f"[*] Factor de ventaja en CPU para el atacante: SHA-256 es {ratio:,.1f}x más rápido que Argon2id.")
    print("    Argon2id impone un costo en memoria (64 MB) y tiempo que neutraliza la viabilidad")
    print("    de ataques de adivinación masivos, mientras que SHA-256 permite millones de cálculos/seg.")
    print("=" * 78)

    # 4. Proyección GPU
    project_gpu_cracking()

    return {
        "sha256": res_sha,
        "argon2id": res_argon,
        "ratio_speedup_sha256": ratio
    }

    return {
        "sha256": res_sha,
        "argon2id": res_argon,
        "ratio_speedup_sha256": ratio
    }


if __name__ == "__main__":
    run_benchmark()
