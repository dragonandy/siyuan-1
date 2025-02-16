import {isNotCtrl, isOnlyMeta} from "./compatibility";
import {Constants} from "../../constants";

// 是否匹配 ⇧⌘[] / ⌘[] / ⌥[] / ⌥⌘[] / ⌥⇧[] / ⌥⇧⌘[] / ⇧[] / []
export const matchHotKey = (hotKey: string, event: KeyboardEvent) => {
    if (!hotKey) {
        return false;
    }

    // []
    if (hotKey.indexOf("⇧") === -1 && hotKey.indexOf("⌘") === -1 && hotKey.indexOf("⌥") === -1 && hotKey.indexOf("⌃") === -1) {
        if (isNotCtrl(event) && !event.altKey && !event.shiftKey && hotKey === Constants.KEYCODELIST[event.keyCode]) {
            return true;
        }
        return false;
    }

    const hotKeys = hotKey.split("");
    if (hotKey.indexOf("F") > -1) {
        hotKeys.forEach((item, index) => {
            if (item === "F") {
                // F1-F12
                hotKeys[index] = "F" + hotKeys.splice(index + 1, 1);
                if (hotKeys[index + 1]) {
                    hotKeys[index + 1] += hotKeys.splice(index + 1, 1);
                }
            }
        });
    }

    // 是否匹配 ⇧[]
    if (hotKey.startsWith("⇧") && hotKeys.length === 2) {
        if (isNotCtrl(event) && !event.altKey && event.shiftKey && hotKeys[1] === Constants.KEYCODELIST[event.keyCode]) {
            return true;
        }
        return false;
    }

    if (hotKey.startsWith("⌥")) {
        let keyCode = hotKeys.length === 3 ? hotKeys[2] : hotKeys[1];
        if (hotKeys.length === 4) {
            keyCode = hotKeys[3];
        }

        const isMatchKey = keyCode === Constants.KEYCODELIST[event.keyCode];
        // 是否匹配 ⌥[] / ⌥⌘[]
        if (isMatchKey && event.altKey && !event.shiftKey &&
            (hotKeys.length === 3 ? (isOnlyMeta(event) && hotKey.startsWith("⌥⌘")) : isNotCtrl(event))) {
            return true;
        }
        // ⌥⇧⌘[]
        if (isMatchKey && hotKey.startsWith("⌥⇧⌘") && hotKeys.length === 4 &&
            event.altKey && event.shiftKey && isOnlyMeta(event)) {
            return true;
        }
        // ⌥⇧[]
        if (isMatchKey && hotKey.startsWith("⌥⇧") && hotKeys.length === 3 &&
            event.altKey && event.shiftKey && isNotCtrl(event)) {
            return true;
        }
        return false;
    }

    // 是否匹配 ⌃[] / ⌃⌘[] / ⌃⌥[] / ⌃⇧[]/ ⌃⌥⇧[]
    if (hotKey.startsWith("⌃")) {
        let keyCode = hotKeys.length === 3 ? hotKeys[2] : hotKeys[1];
        if (hotKeys.length === 4) {
            keyCode = hotKeys[3];
        } else if (hotKeys.length === 5) {
            keyCode = hotKeys[4];
        }

        const isMatchKey = keyCode === Constants.KEYCODELIST[event.keyCode];
        // 是否匹配 ⌃[] / ⌃⌘[]
        if (isMatchKey && event.ctrlKey && !event.altKey && !event.shiftKey && hotKeys.length < 4 &&
            (hotKeys.length === 3 ? (event.metaKey && hotKey.startsWith("⌃⌘")) : !event.metaKey)) {
            return true;
        }
        // ⌃⇧[]
        if (isMatchKey && hotKey.startsWith("⌃⇧") && hotKeys.length === 3 &&
            event.ctrlKey && !event.altKey && event.shiftKey && !event.metaKey) {
            return true;
        }
        // ⌃⌥[]
        if (isMatchKey && hotKey.startsWith("⌃⌥") && hotKeys.length === 3 &&
            event.ctrlKey && event.altKey && !event.shiftKey && !event.metaKey) {
            return true;
        }
        // ⌃⌥⇧[] / ⌃⌥⌘[] / ⌃⇧⌘[]
        if (isMatchKey && hotKeys.length === 4 && event.ctrlKey &&
            (
                (hotKey.startsWith("⌃⌥⇧") && event.shiftKey && !event.metaKey && event.altKey) ||
                (hotKey.startsWith("⌃⌥⌘") && !event.shiftKey && event.metaKey && event.altKey) ||
                (hotKey.startsWith("⌃⇧⌘") && event.shiftKey && event.metaKey && !event.altKey)
            )
        ) {
            return true;
        }

        // ⌃⌥⇧⌘[]
        if (isMatchKey && hotKeys.length === 5 && event.ctrlKey && event.shiftKey && event.metaKey && event.altKey) {
            return true;
        }
        return false;
    }

    // 是否匹配 ⇧⌘[] / ⌘[]
    const hasShift = hotKeys.length > 2 && (hotKeys[0] === "⇧");
    if (isOnlyMeta(event) && !event.altKey && ((!hasShift && !event.shiftKey) || (hasShift && event.shiftKey))) {
        return (hasShift ? hotKeys[2] : hotKeys[1]) === Constants.KEYCODELIST[event.keyCode];
    }
    return false;
};

