package vpxencoder

import (
	"fmt"
	"unsafe"
)

import "C"

const chanSize = 2

func (v *VpxEncoder) init() error {
	v.frameCount = 0

	codecName := C.CString("vp8")
	encoder := C.get_vpx_encoder_by_name(codecName)
	C.free(unsafe.Pointer(codecName))

	if encoder == nil {
		return fmt.Errorf("get_vpx_encoder_by_name failed")
	}
	if C.vpx_img_alloc(&v.vpxImage, C.VPX_IMG_FMT_I420, v.width, v.height, 0) == nil {
		return fmt.Errorf("vpx_img_alloc failed")
	}

	var cfg C.vpx_codec_enc_cfg_t
	if C.call_vpx_codec_enc_config_default(encoder, &cfg) != 0 {
		return fmt.Errorf("Failed to get default codec config")
	}
	cfg.g_w = v.width
	cfg.g_h = v.height
	cfg.g_timebase.num = 1
	cfg.g_timebase.den = v.fps
	cfg.rc_target_bitrate = v.bitrate
	cfg.g_error_resilient = 1

	if C.call_vpx_codec_enc_init(&v.vpxCodexCtx, encoder, &cfg) != 0 {
		return fmt.Errorf("Failed to initialize encoder")
	}
	v.started = true
	go v.startLooping()
	return nil
}

func (v *VpxEncoder) startLooping() {
	go func() {
		for {
			yuv := <-v.Input
			// Add Image
			v.vpxCodexIter = nil
			C.vpx_img_read(&v.vpxImage, unsafe.Pointer(&yuv[0]))

			var flags C.int
			if v.keyFrameInterval > 0 && v.frameCount%v.keyFrameInterval == 0 {
				flags |= C.VPX_EFLAG_FORCE_KF
			}
			if C.vpx_codec_encode(&v.vpxCodexCtx, &v.vpxImage, C.vpx_codec_pts_t(v.frameCount), 1, C.vpx_enc_frame_flags_t(flags), C.VPX_DL_REALTIME) != 0 {
				fmt.Println("Failed to encode frame")
			}
			v.frameCount++

			// Get Frame
			for {
				goBytes := C.get_frame_buffer(&v.vpxCodexCtx, &v.vpxCodexIter)
				if goBytes.bs == nil {
					break
				}
				bs := C.GoBytes(goBytes.bs, goBytes.size)
				// if buffer is full skip frame
				if len(v.Output) >= cap(v.Output) {
					continue
				}
				v.Output <- bs
			}
		}
	}()
}

// NewVpxEncoder create vp8 encoder
func NewVpxEncoder(w, h, fps, bitrate, keyframe int) (*VpxEncoder, error) {
	v := &VpxEncoder{
		Output: make(chan []byte, 5*chanSize),
		Input:  make(chan []byte, chanSize),
		// C
		width:            C.uint(w),
		height:           C.uint(h),
		fps:              C.int(fps),
		bitrate:          C.uint(bitrate),
		keyFrameInterval: C.int(keyframe),
		frameCount:       C.int(0),
	}

	if err := v.init(); err != nil {
		return nil, err
	}

	return v, nil
}

// VpxEncoder yuvI420 image to vp8 video
type VpxEncoder struct {
	started bool
	Output  chan []byte // frame
	Input   chan []byte // yuvI420
	// C
	width            C.uint
	height           C.uint
	fps              C.int
	bitrate          C.uint
	keyFrameInterval C.int
	frameCount       C.int
	vpxCodexCtx      C.vpx_codec_ctx_t
	vpxImage         C.vpx_image_t
	vpxCodexIter     C.vpx_codec_iter_t
}


// Release memory and stop loop
func (v *VpxEncoder) Release() {
	v.started = false
	if v.started {
		close(v.Input)
		close(v.Output)
		C.vpx_img_free(&v.vpxImage)
		C.vpx_codec_destroy(&v.vpxCodexCtx)
	}
}
