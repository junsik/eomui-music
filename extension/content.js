// ============================================================
// 어무이 음악 다운로더 - Content Script (시니어 친화적)
// 원칙: 클릭 한 번 → 바탕화면에 MP3 저장. 복사/붙여넣기 불필요.
// 버튼 위치: YouTube 영상 하단 액션바 (좋아요/공유 버튼 옆)
// ============================================================

(function () {
  "use strict";

  const SERVER_URL = "http://localhost:8080";
  const BUTTON_ID = "eomui-mp3-button";
  const STATUS_ID = "eomui-mp3-status";

  // ===== 상태 관리 =====
  let isDownloading = false;
  let injectionTimer = null;

  // ===== 기본 아이콘 (음표) =====
  const MUSIC_ICON = `
    <svg viewBox="0 0 24 24" fill="currentColor">
      <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
    </svg>
  `;
  const SUCCESS_ICON = `
    <svg viewBox="0 0 24 24" fill="currentColor">
      <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
    </svg>
  `;
  const ERROR_ICON = `
    <svg viewBox="0 0 24 24" fill="currentColor">
      <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
    </svg>
  `;

  // ===== YouTube 액션바 찾기 (좋아요/공유 버튼이 있는 곳) =====
  // 여러 셀렉터를 시도 - YouTube 버전별로 마크업이 다름
  function findActionBar() {
    // 우선순위별 셀렉터 목록
    const selectors = [
      "#top-level-buttons-computed",   // 최신 YouTube (Like/Dislike/Share 컨테이너)
      "#top-level-buttons",            // 구버전
      "ytd-menu-renderer #top-level-buttons-computed",
      "#actions-inner #top-level-buttons-computed",
      "#actions #top-level-buttons-computed",
      "ytd-watch-metadata #top-level-buttons-computed",
    ];

    for (const sel of selectors) {
      const el = document.querySelector(sel);
      if (el && el.isConnected) {
        // 실제 버튼들이 보이는지 확인 (화면에 노출되는 액션바인지)
        const rect = el.getBoundingClientRect();
        if (rect.width > 50) {
          return el;
        }
      }
    }

    // 폴백: 액션 영역 전체
    const fallbackSelectors = [
      "#actions-inner",
      "#actions",
      "ytd-watch-metadata #actions-inner",
    ];
    for (const sel of fallbackSelectors) {
      const el = document.querySelector(sel);
      if (el && el.isConnected) {
        const rect = el.getBoundingClientRect();
        if (rect.width > 100) {
          return el;
        }
      }
    }

    return null;
  }

  // ===== MP3 버튼 생성 =====
  function createButton() {
    if (document.getElementById(BUTTON_ID)) {
      // 이미 존재하면 연결 상태만 확인
      const existing = document.getElementById(BUTTON_ID);
      if (existing.isConnected) return;
      existing.remove();
    }

    const btn = document.createElement("button");
    btn.id = BUTTON_ID;
    btn.type = "button";
    btn.className = "eomui-action-btn";
    btn.setAttribute("aria-label", "MP3 다운로드");
    btn.innerHTML = `${MUSIC_ICON}<span class="eomui-btn-label">MP3 다운로드</span>`;
    btn.addEventListener("click", onDownloadClick);
    return btn;
  }

  // ===== 버튼을 YouTube 액션바에 삽입 =====
  function injectButton() {
    // YouTube 영상 페이지가 아니면 시도하지 않음
    const url = window.location.href;
    const isWatchPage = /youtube\.com\/(watch\?v=|shorts\/)|youtu\.be\//.test(url);
    if (!isWatchPage) return false;

    const actionBar = findActionBar();
    if (!actionBar) return false;

    // 이미 있으면 스킵
    if (document.getElementById(BUTTON_ID)?.isConnected) return true;

    const btn = createButton();
    if (!btn) return false;

    // 액션바 끝에 삽입 (공유/저장 버튼 뒤)
    actionBar.appendChild(btn);

    // 상태 표시 오버레이 (버튼 상단에 떠 있도록 body에 추가)
    if (!document.getElementById(STATUS_ID)) {
      const overlay = document.createElement("div");
      overlay.id = STATUS_ID;
      overlay.className = "eomui-hidden";
      document.body.appendChild(overlay);
    }

    return true;
  }

  // ===== 상태 표시 오버레이 =====
  function showStatus(type, message) {
    const el = document.getElementById(STATUS_ID);
    if (!el) return;
    el.className = "eomui-visible eomui-" + type;
    el.innerHTML = message;
    if (type !== "loading") {
      setTimeout(() => {
        el.className = "eomui-hidden";
      }, 6000);
    }
  }

  function hideStatus() {
    const el = document.getElementById(STATUS_ID);
    if (el) el.className = "eomui-hidden";
  }

  // ===== 버튼 상태 업데이트 =====
  function setButtonState(state) {
    const btn = document.getElementById(BUTTON_ID);
    if (!btn) return;
    btn.classList.remove("loading", "success", "error");

    if (state === "loading") {
      btn.classList.add("loading");
      btn.innerHTML = `<div class="eomui-spinner"></div><span class="eomui-btn-label">다운로드 중...</span>`;
      btn.disabled = true;
    } else if (state === "success") {
      btn.classList.add("success");
      btn.innerHTML = `${SUCCESS_ICON}<span class="eomui-btn-label">완료!</span>`;
      btn.disabled = false;
      setTimeout(() => {
        if (btn.isConnected) {
          btn.classList.remove("success");
          btn.innerHTML = `${MUSIC_ICON}<span class="eomui-btn-label">MP3 다운로드</span>`;
        }
      }, 4000);
    } else if (state === "error") {
      btn.classList.add("error");
      btn.innerHTML = `${ERROR_ICON}<span class="eomui-btn-label">다시 시도</span>`;
      btn.disabled = false;
    } else {
      btn.innerHTML = `${MUSIC_ICON}<span class="eomui-btn-label">MP3 다운로드</span>`;
      btn.disabled = false;
    }
  }

  // ===== 다운로드 클릭 핸들러 (단 한 번의 클릭) =====
  async function onDownloadClick() {
    if (isDownloading) return;

    const videoUrl = window.location.href.split("&")[0];
    console.log("[어무이] 다운로드 시작:", videoUrl);

    // YouTube 영상 페이지인지 확인
    const isValid = /youtube\.com\/(watch\?v=|shorts\/)|youtu\.be\//.test(videoUrl);
    if (!isValid) {
      showStatus("error", "<b>사용할 수 없는 페이지</b><small>YouTube 영상 페이지에서만 다운로드할 수 있습니다.</small>");
      setButtonState("error");
      return;
    }

    isDownloading = true;
    setButtonState("loading");
    showStatus("loading", "<b>음악을 준비하고 있어요</b><small>잠시만 기다려 주세요. 긴 영상은 시간이 더 걸려요.</small>");

    try {
      const resp = await fetch(`${SERVER_URL}/api/download?url=${encodeURIComponent(videoUrl)}`, {
        method: "GET",
        // 서버가 이 헤더로 우리 확장의 요청인지 구분한다.
        // 다른 웹사이트는 <img>/<script> 같은 단순 요청에 헤더를 붙일 수 없다.
        headers: { "X-Eomui-Client": "1" },
      });

      const data = await resp.json().catch(() => ({ error: `서버 응답 오류 (${resp.status})` }));

      if (!resp.ok) {
        throw new Error(data.error || `서버 오류 (${resp.status})`);
      }

      setButtonState("success");
      const filename = data.filename || "음악.mp3";

      if (data.duplicate) {
        // 같은 영상을 또 누른 경우. 다시 받지 않았다는 것을 분명히 알린다.
        showStatus(
          "success",
          `<b>이미 받으신 노래예요</b><small>${escapeHtml(filename)}</small><small style="opacity:0.7">다시 받지 않았어요.</small>`
        );
        return;
      }

      showStatus(
        "success",
        `<b>다운로드 완료!</b><small>${escapeHtml(filename)}</small><small style="opacity:0.7">바탕화면에 저장되었어요.</small>`
      );
    } catch (err) {
      console.error("[어무이] 다운로드 오류:", err);
      const msg = err.message || "다운로드에 실패했습니다.";

      let userMsg = msg;
      if (msg.includes("Failed to fetch") || msg.includes("NetworkError")) {
        userMsg = "<b>프로그램이 켜져 있지 않아요</b><small>어무이 음악 다운로더 프로그램을 실행해 주세요.</small>";
      } else if (msg.includes("Sign in")) {
        userMsg = "<b>로그인이 필요해요</b><small>이 영상은 로그인해야 다운로드할 수 있어요.</small>";
      } else if (msg.includes("Private") || msg.includes("unavailable")) {
        userMsg = "<b>사용할 수 없는 영상</b><small>비공개이거나 삭제된 영상이에요.</small>";
      } else if (msg.includes("429")) {
        userMsg = "<b>잠시만 기다려 주세요</b><small>요청이 너무 많아요. 잠시 후 다시 시도해 주세요.</small>";
      } else {
        userMsg = `<b>다운로드 실패</b><small>${escapeHtml(msg)}</small>`;
      }

      setButtonState("error");
      showStatus("error", userMsg);
    } finally {
      isDownloading = false;
    }
  }

  // ===== HTML 이스케이프 (파일명 표시용) =====
  function escapeHtml(str) {
    if (!str) return "";
    return String(str)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#039;");
  }

  // ===== 초기화: 액션바를 찾을 때까지 재시도 =====
  function tryInject(retriesLeft = 30) {
    if (injectButton()) return;

    if (retriesLeft > 0) {
      clearTimeout(injectionTimer);
      injectionTimer = setTimeout(() => tryInject(retriesLeft - 1), 500);
    }
  }

  function init() {
    if (!document.body) return;
    tryInject(30);
  }

  // 최초 실행
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  // ===== YouTube SPA URL 변경 감지 =====
  // YouTube는 SPA이므로 페이지 전환 시 액션바가 새로 그려짐 → 버튼 재삽입
  let lastUrl = window.location.href;
  const observer = new MutationObserver(() => {
    if (window.location.href !== lastUrl) {
      lastUrl = window.location.href;
      // 페이지 전환 후 액션바가 나타날 때까지 재시도
      setTimeout(() => {
        tryInject(30);
        setButtonState("idle");
        hideStatus();
      }, 800);
    } else {
      // 같은 URL이어도 버튼이 사라졌으면 다시 삽입
      const btn = document.getElementById(BUTTON_ID);
      if (!btn || !btn.isConnected) {
        tryInject(5);
      }
    }
  });

  if (document.body) {
    observer.observe(document.body, { childList: true, subtree: true });
  } else {
    document.addEventListener("DOMContentLoaded", () => {
      observer.observe(document.body, { childList: true, subtree: true });
    });
  }
})();
