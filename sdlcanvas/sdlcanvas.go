package sdlcanvas

import (
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"runtime"
	"time"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/tfriedel6/canvas"
	"github.com/tfriedel6/canvas/goglimpl"
	"github.com/veandco/go-sdl2/sdl"
)

// Window represents the opened window with GL context. The Mouse* and Key*
// functions can be set for callbacks
type Window struct {
	Window     *sdl.Window
	GLContext  sdl.GLContext
	frameTimes [10]time.Time
	frameIndex int
	frameCount int
	fps        float32
	close      bool
	events     []sdl.Event
	Event      func(event sdl.Event)
	MouseDown  func(button, x, y int)
	MouseMove  func(x, y int)
	MouseUp    func(button, x, y int)
	KeyDown    func(scancode int, rune rune, name string)
	KeyUp      func(scancode int, rune rune, name string)
}

// CreateWindow creates a window using SDL and initializes the OpenGL context
func CreateWindow(w, h int, title string) (*Window, *canvas.Canvas, error) {
	runtime.LockOSThread()

	// init SDL
	err := sdl.Init(sdl.INIT_VIDEO)
	if err != nil {
		return nil, nil, fmt.Errorf("Error initializing SDL: %v", err)
	}

	sdl.GL_SetAttribute(sdl.GL_RED_SIZE, 8)
	sdl.GL_SetAttribute(sdl.GL_GREEN_SIZE, 8)
	sdl.GL_SetAttribute(sdl.GL_BLUE_SIZE, 8)
	sdl.GL_SetAttribute(sdl.GL_ALPHA_SIZE, 8)
	sdl.GL_SetAttribute(sdl.GL_DEPTH_SIZE, 0)
	sdl.GL_SetAttribute(sdl.GL_STENCIL_SIZE, 8)
	sdl.GL_SetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	sdl.GL_SetAttribute(sdl.GL_MULTISAMPLEBUFFERS, 1)
	sdl.GL_SetAttribute(sdl.GL_MULTISAMPLESAMPLES, 4)

	// create window
	window, err := sdl.CreateWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, w, h, sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
	if err != nil {
		sdl.GL_SetAttribute(sdl.GL_MULTISAMPLEBUFFERS, 0)
		sdl.GL_SetAttribute(sdl.GL_MULTISAMPLESAMPLES, 0)
		window, err = sdl.CreateWindow(title, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, w, h, sdl.WINDOW_RESIZABLE|sdl.WINDOW_OPENGL)
		if err != nil {
			return nil, nil, fmt.Errorf("Error creating window: %v", err)
		}
	}

	// create GL context
	glContext, err := sdl.GL_CreateContext(window)
	if err != nil {
		return nil, nil, fmt.Errorf("Error creating GL context: %v", err)
	}

	// init GL
	err = gl.Init()
	if err != nil {
		return nil, nil, fmt.Errorf("Error initializing GL: %v", err)
	}

	sdl.GL_SetSwapInterval(1)
	gl.Enable(gl.MULTISAMPLE)

	err = canvas.LoadGL(goglimpl.GLImpl{})
	if err != nil {
		return nil, nil, fmt.Errorf("Error loading canvas GL assets: %v", err)
	}

	cv := canvas.New(0, 0, w, h)
	wnd := &Window{
		Window:    window,
		GLContext: glContext,
		events:    make([]sdl.Event, 0, 100),
	}

	return wnd, cv, nil
}

// Destroy destroys the GL context and the window
func (wnd *Window) Destroy() {
	sdl.GL_DeleteContext(wnd.GLContext)
	wnd.Window.Destroy()
}

// FPS returns the frames per second (averaged over 10 frames)
func (wnd *Window) FPS() float32 {
	return wnd.fps
}

// Close can be used to end a call to MainLoop
func (wnd *Window) Close() {
	wnd.close = true
}

// StartFrame handles events and gets the window ready for rendering
func (wnd *Window) StartFrame() error {
	err := sdl.GL_MakeCurrent(wnd.Window, wnd.GLContext)
	if err != nil {
		return err
	}

	wnd.events = wnd.events[:0]
	for {
		event := sdl.PollEvent()
		if event == nil {
			break
		}

		handled := false
		switch e := event.(type) {
		case *sdl.MouseButtonEvent:
			if e.Type == sdl.MOUSEBUTTONDOWN {
				if wnd.MouseDown != nil {
					wnd.MouseDown(int(e.Button), int(e.X), int(e.Y))
					handled = true
				}
			} else if e.Type == sdl.MOUSEBUTTONUP {
				if wnd.MouseUp != nil {
					wnd.MouseUp(int(e.Button), int(e.X), int(e.Y))
					handled = true
				}
			}
		case *sdl.MouseMotionEvent:
			if wnd.MouseMove != nil {
				wnd.MouseMove(int(e.X), int(e.Y))
				handled = true
			}
		case *sdl.KeyDownEvent:
			if wnd.KeyDown != nil {
				wnd.KeyDown(int(e.Keysym.Scancode), rune(e.Keysym.Unicode), keyName(e.Keysym.Scancode))
				handled = true
			}
		case *sdl.KeyUpEvent:
			if wnd.KeyUp != nil {
				wnd.KeyUp(int(e.Keysym.Scancode), rune(e.Keysym.Unicode), keyName(e.Keysym.Scancode))
				handled = true
			}
		}

		if !handled && wnd.Event != nil {
			wnd.Event(event)
			handled = true
		}

		if !handled {
			wnd.events = append(wnd.events, event)
		}
	}

	return nil
}

// FinishFrame updates the FPS count and displays the frame
func (wnd *Window) FinishFrame() {
	now := time.Now()
	wnd.frameTimes[wnd.frameIndex] = now
	wnd.frameIndex++
	wnd.frameIndex %= len(wnd.frameTimes)
	if wnd.frameCount < len(wnd.frameTimes) {
		wnd.frameCount++
	} else {
		diff := now.Sub(wnd.frameTimes[wnd.frameIndex]).Seconds()
		wnd.fps = float32(wnd.frameCount-1) / float32(diff)
	}

	sdl.GL_SwapWindow(wnd.Window)
}

// MainLoop runs a main loop and calls run on every frame
func (wnd *Window) MainLoop(run func()) {
	// main loop
	for !wnd.close {
		err := wnd.StartFrame()
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		for _, event := range wnd.events {
			switch e := event.(type) {
			case *sdl.WindowEvent:
				if e.Event == sdl.WINDOWEVENT_CLOSE {
					wnd.close = true
				}
			case *sdl.KeyDownEvent:
				if e.Keysym.Scancode == sdl.SCANCODE_ESCAPE {
					wnd.close = true
				}
			}
		}

		run()

		wnd.FinishFrame()
	}
}
