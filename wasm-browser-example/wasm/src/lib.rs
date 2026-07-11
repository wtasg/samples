use wasm_bindgen::prelude::*;

// ── Fibonacci ──────────────────────────────────────────────────────────────
/// Compute the nth Fibonacci number using a fast iterative algorithm.
/// Returns a u64 (capped at u64::MAX to avoid overflow for large n).
#[wasm_bindgen]
pub fn fibonacci(n: u32) -> u64 {
    if n == 0 {
        return 0;
    }
    if n == 1 {
        return 1;
    }
    let mut a: u64 = 0;
    let mut b: u64 = 1;
    for _ in 2..=n {
        let (next, overflowed) = a.overflowing_add(b);
        if overflowed {
            return u64::MAX;
        }
        a = b;
        b = next;
    }
    b
}

// ── Mandelbrot ─────────────────────────────────────────────────────────────
/// Render a Mandelbrot set into a flat RGBA Uint8Array.
/// Returns a Vec<u8> of length width * height * 4.
/// Parameters: width, height in pixels; max_iter = maximum iterations (200–1000 typical).
/// The viewport is fixed to the classic Mandelbrot view: re ∈ [-2.5, 1.0], im ∈ [-1.25, 1.25].
#[wasm_bindgen]
pub fn mandelbrot(width: u32, height: u32, max_iter: u32) -> Vec<u8> {
    let mut pixels = vec![0u8; (width * height * 4) as usize];

    let re_min: f64 = -2.5;
    let re_max: f64 = 1.0;
    let im_min: f64 = -1.25;
    let im_max: f64 = 1.25;

    for py in 0..height {
        for px in 0..width {
            let c_re = re_min + (px as f64 / width  as f64) * (re_max - re_min);
            let c_im = im_min + (py as f64 / height as f64) * (im_max - im_min);

            let mut z_re = 0.0f64;
            let mut z_im = 0.0f64;
            let mut iter = 0u32;

            while iter < max_iter {
                let z_re2 = z_re * z_re;
                let z_im2 = z_im * z_im;
                if z_re2 + z_im2 > 4.0 {
                    break;
                }
                z_im = 2.0 * z_re * z_im + c_im;
                z_re = z_re2 - z_im2 + c_re;
                iter += 1;
            }

            let base = ((py * width + px) * 4) as usize;
            if iter == max_iter {
                // Inside the set → black
                pixels[base]     = 0;
                pixels[base + 1] = 0;
                pixels[base + 2] = 0;
                pixels[base + 3] = 255;
            } else {
                // Smooth coloring: map escape iteration to HSL-like spectrum
                let t = iter as f64 / max_iter as f64;
                let (r, g, b) = palette(t, iter);
                pixels[base]     = r;
                pixels[base + 1] = g;
                pixels[base + 2] = b;
                pixels[base + 3] = 255;
            }
        }
    }

    pixels
}

/// Map a normalized escape value t ∈ [0,1] to an RGB colour.
fn palette(t: f64, iter: u32) -> (u8, u8, u8) {
    // Vivid ultra-fractal style palette
    let smooth_t = t + 1.0 - (iter as f64).ln().ln() / (2.0f64).ln();
    let angle = smooth_t * 6.28318; // 2π cycle

    let r = (0.5 + 0.5 * (angle            ).cos() * 255.0) as u8;
    let g = (0.5 + 0.5 * (angle + 2.094395 ).cos() * 255.0) as u8; // +2π/3
    let b = (0.5 + 0.5 * (angle + 4.188790 ).cos() * 255.0) as u8; // +4π/3
    (r, g, b)
}

// ── FNV-1a Hash ────────────────────────────────────────────────────────────
/// Compute a 64-bit FNV-1a hash of a string. Returns the hash as a decimal string
/// (JS BigInt / number handles it via the string representation).
#[wasm_bindgen]
pub fn fnv1a_hash(s: &str) -> String {
    const FNV_OFFSET: u64 = 14695981039346656037;
    const FNV_PRIME:  u64 = 1099511628211;

    let mut hash = FNV_OFFSET;
    for byte in s.bytes() {
        hash ^= byte as u64;
        hash = hash.wrapping_mul(FNV_PRIME);
    }
    format!("{}", hash)
}

// ── Prime Sieve ────────────────────────────────────────────────────────────
/// Count primes up to n using the Sieve of Eratosthenes.
/// Returns the count. Used for the compute benchmark.
#[wasm_bindgen]
pub fn count_primes(n: u32) -> u32 {
    if n < 2 {
        return 0;
    }
    let n = n as usize;
    let mut sieve = vec![true; n + 1];
    sieve[0] = false;
    sieve[1] = false;

    let mut i = 2;
    while i * i <= n {
        if sieve[i] {
            let mut j = i * i;
            while j <= n {
                sieve[j] = false;
                j += i;
            }
        }
        i += 1;
    }
    sieve.iter().filter(|&&v| v).count() as u32
}

// ── Unit Tests ─────────────────────────────────────────────────────────────
#[cfg(test)]
mod tests {
    use super::*;

    // Fibonacci
    #[test]
    fn test_fibonacci_base_cases() {
        assert_eq!(fibonacci(0), 0);
        assert_eq!(fibonacci(1), 1);
    }

    #[test]
    fn test_fibonacci_sequence() {
        assert_eq!(fibonacci(2),  1);
        assert_eq!(fibonacci(5),  5);
        assert_eq!(fibonacci(10), 55);
        assert_eq!(fibonacci(20), 6765);
        assert_eq!(fibonacci(40), 102334155);
    }

    // Mandelbrot
    #[test]
    fn test_mandelbrot_output_size() {
        let pixels = mandelbrot(16, 8, 64);
        assert_eq!(pixels.len(), 16 * 8 * 4);
    }

    #[test]
    fn test_mandelbrot_alpha_is_255() {
        let pixels = mandelbrot(8, 8, 64);
        for i in 0..(8 * 8) {
            assert_eq!(pixels[i * 4 + 3], 255, "Alpha channel should always be 255");
        }
    }

    #[test]
    fn test_mandelbrot_origin_is_black() {
        // The origin (0,0) is inside the Mandelbrot set — it should render as black.
        // With our viewport re∈[-2.5,1.0] im∈[-1.25,1.25], center pixel (w/2, h/2)
        // maps to approx (c_re=-0.75, c_im=0) which is inside the set.
        let w: u32 = 100;
        let h: u32 = 100;
        let pixels = mandelbrot(w, h, 256);
        let cx = (w / 2) as usize;
        let cy = (h / 2) as usize;
        let base = (cy * w as usize + cx) * 4;
        // Inside-set pixels are black (0,0,0,255)
        assert_eq!(pixels[base], 0, "Center should be R=0 (inside set)");
        assert_eq!(pixels[base + 1], 0, "Center should be G=0 (inside set)");
        assert_eq!(pixels[base + 2], 0, "Center should be B=0 (inside set)");
    }

    // FNV-1a Hash
    #[test]
    fn test_fnv1a_empty_string() {
        // FNV-1a of "" = 14695981039346656037
        let h = fnv1a_hash("");
        assert_eq!(h, "14695981039346656037");
    }

    #[test]
    fn test_fnv1a_known_value() {
        // FNV-1a of "hello" = 11831194018420276491
        let h = fnv1a_hash("hello");
        assert_eq!(h, "11831194018420276491");
    }

    #[test]
    fn test_fnv1a_different_strings_give_different_hashes() {
        assert_ne!(fnv1a_hash("foo"), fnv1a_hash("bar"));
    }

    // Prime sieve
    #[test]
    fn test_count_primes_small() {
        assert_eq!(count_primes(0), 0);
        assert_eq!(count_primes(1), 0);
        assert_eq!(count_primes(2), 1);
        assert_eq!(count_primes(10), 4); // 2, 3, 5, 7
    }

    #[test]
    fn test_count_primes_100() {
        assert_eq!(count_primes(100), 25);
    }

    #[test]
    fn test_count_primes_1000() {
        assert_eq!(count_primes(1000), 168);
    }

    #[test]
    fn test_count_primes_1_000_000() {
        assert_eq!(count_primes(1_000_000), 78498);
    }
}
