import { api } from './client.js'

const DEFAULT_CHUNK_SIZE = 50 * 1024 * 1024 // 50 MB
const MAX_RETRIES = 3

async function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * Uploads a file in chunks to the backend.
 *
 * @param {File} file - The file to upload.
 * @param {Object} metadata - Passed to completeUpload: { name, effect_type, ... }
 * @param {Object} options
 * @param {function} [options.onProgress] - Called with { percent, chunkIndex, totalChunks }
 * @param {number}   [options.chunkSize]  - Chunk size in bytes, default 50 MB
 * @param {AbortSignal} [options.signal]  - Optional AbortSignal to cancel the upload
 * @returns {Promise<Object>} Result from completeUpload
 */
export async function uploadFileChunked(file, metadata, { onProgress, chunkSize = DEFAULT_CHUNK_SIZE, signal } = {}) {
  const { upload_id: uploadId } = await api.initUpload()

  const totalChunks = Math.ceil(file.size / chunkSize)

  try {
    for (let i = 0; i < totalChunks; i++) {
      if (signal?.aborted) {
        throw new DOMException('Upload aborted', 'AbortError')
      }

      const start = i * chunkSize
      const end = Math.min(start + chunkSize, file.size)
      const blob = file.slice(start, end)

      let lastError
      for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
        if (signal?.aborted) {
          throw new DOMException('Upload aborted', 'AbortError')
        }
        try {
          await api.uploadChunk(uploadId, i, blob)
          lastError = null
          break
        } catch (err) {
          lastError = err
          if (attempt < MAX_RETRIES - 1) {
            await sleep(1000 * Math.pow(2, attempt)) // 1s, 2s, 4s
          }
        }
      }

      if (lastError) {
        throw lastError
      }

      if (onProgress) {
        onProgress({
          percent: Math.round(((i + 1) / totalChunks) * 100),
          chunkIndex: i,
          totalChunks,
        })
      }
    }

    const result = await api.completeUpload(uploadId, metadata)
    return result
  } catch (err) {
    // Best-effort abort — don't let cleanup error mask the original
    try {
      await api.abortUpload(uploadId)
    } catch (_) { /* ignore */ }
    throw err
  }
}
