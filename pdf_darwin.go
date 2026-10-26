package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework WebKit

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include <stdlib.h>

@interface RMDelegate : NSObject <WKNavigationDelegate>
@property (nonatomic) BOOL loaded;
@property (nonatomic) BOOL printed;
@property (nonatomic) BOOL printSucceeded;
@property (nonatomic, strong) NSError *error;
@end

@implementation RMDelegate
- (void)webView:(WKWebView *)webView didFinishNavigation:(WKNavigation *)navigation {
	self.loaded = YES;
}

- (void)webView:(WKWebView *)webView didFailNavigation:(WKNavigation *)navigation withError:(NSError *)error {
	self.error = error;
	self.loaded = YES;
}

- (void)webView:(WKWebView *)webView didFailProvisionalNavigation:(WKNavigation *)navigation withError:(NSError *)error {
	self.error = error;
	self.loaded = YES;
}

- (void)printOperationDidRun:(NSPrintOperation *)op success:(BOOL)success contextInfo:(void *)info {
	self.printSucceeded = success;
	self.printed = YES;
}
@end

static BOOL rmSpin(BOOL (^cond)(void), NSDate *deadline) {
	while (!cond()) {
		if ([deadline timeIntervalSinceNow] <= 0) {
			return NO;
		}
		@autoreleasepool {
			NSEvent *ev = [NSApp nextEventMatchingMask:NSEventMaskAny
			                                 untilDate:[NSDate dateWithTimeIntervalSinceNow:0.05]
			                                    inMode:NSDefaultRunLoopMode
			                                   dequeue:YES];
			if (ev) {
				[NSApp sendEvent:ev];
			}
		}
	}
	return YES;
}

static char *rmRenderPDF(const char *htmlPath, const char *outPath, double timeout, double settle) {
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp setActivationPolicy:NSApplicationActivationPolicyProhibited];

		NSDate *deadline = [NSDate dateWithTimeIntervalSinceNow:timeout];
		RMDelegate *delegate = [RMDelegate new];

		WKWebViewConfiguration *config = [WKWebViewConfiguration new];
		config.preferences.shouldPrintBackgrounds = YES;

		NSRect frame = NSMakeRect(0, 0, 800, 1000);
		WKWebView *webView = [[WKWebView alloc] initWithFrame:frame configuration:config];
		webView.navigationDelegate = delegate;

		NSWindow *window = [[NSWindow alloc] initWithContentRect:frame
		                                               styleMask:NSWindowStyleMaskBorderless
		                                                 backing:NSBackingStoreBuffered
		                                                   defer:NO];
		window.releasedWhenClosed = NO;
		window.contentView = webView;

		// 로컬 이미지를 읽을 수 있도록 파일 시스템 전체에 읽기 권한을 준다
		[webView loadFileURL:[NSURL fileURLWithPath:[NSString stringWithUTF8String:htmlPath]]
		 allowingReadAccessToURL:[NSURL fileURLWithPath:@"/" isDirectory:YES]];
		if (!rmSpin(^{ return delegate.loaded; }, deadline)) {
			return strdup("timed out loading page");
		}
		if (delegate.error) {
			return strdup(delegate.error.localizedDescription.UTF8String);
		}

		// Mermaid 렌더링 대기
		rmSpin(^{ return NO; }, [[NSDate dateWithTimeIntervalSinceNow:settle] earlierDate:deadline]);

		NSPrintInfo *info = [[NSPrintInfo alloc] initWithDictionary:@{
			NSPrintJobDisposition: NSPrintSaveJob,
			NSPrintJobSavingURL: [NSURL fileURLWithPath:[NSString stringWithUTF8String:outPath]],
		}];
		info.paperSize = NSMakeSize(595.44, 841.68); // A4
		info.topMargin = 43.2;                       // 0.6in
		info.bottomMargin = 43.2;
		info.leftMargin = 43.2;
		info.rightMargin = 43.2;
		info.horizontalPagination = NSPrintingPaginationModeAutomatic;
		info.verticalPagination = NSPrintingPaginationModeAutomatic;
		info.horizontallyCentered = NO;
		info.verticallyCentered = NO;

		NSPrintOperation *op = [webView printOperationWithPrintInfo:info];
		op.showsPrintPanel = NO;
		op.showsProgressPanel = NO;
		// WKWebView의 인쇄 뷰는 frame이 비어 있으면 빈 페이지를 출력한다
		op.view.frame = webView.bounds;
		[op runOperationModalForWindow:window
		                      delegate:delegate
		                didRunSelector:@selector(printOperationDidRun:success:contextInfo:)
		                   contextInfo:NULL];
		if (!rmSpin(^{ return delegate.printed; }, deadline)) {
			return strdup("timed out printing PDF");
		}
		if (!delegate.printSucceeded) {
			return strdup("print operation failed");
		}
		return NULL;
	}
}
*/
import "C"

import (
	"errors"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

func init() {
	// Cocoa(WebKit)는 메인 스레드에서만 동작한다
	runtime.LockOSThread()
}

func renderPDF(htmlContent, baseDir string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "rendermd-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	// 상대 경로 이미지가 입력 파일 기준으로 해석되도록 <base>를 넣는다
	baseURL := (&url.URL{Scheme: "file", Path: baseDir + "/"}).String()
	htmlContent = strings.Replace(htmlContent, "<head>", `<head>
    <base href="`+html.EscapeString(baseURL)+`">`, 1)

	htmlPath := filepath.Join(dir, "page.html")
	if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
		return nil, err
	}
	outPath := filepath.Join(dir, "out.pdf")

	cHTML := C.CString(htmlPath)
	defer C.free(unsafe.Pointer(cHTML))
	cPath := C.CString(outPath)
	defer C.free(unsafe.Pointer(cPath))

	if cErr := C.rmRenderPDF(cHTML, cPath, 30, 2); cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return nil, errors.New(C.GoString(cErr))
	}

	return os.ReadFile(outPath)
}
