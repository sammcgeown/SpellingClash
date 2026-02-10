(function () {
    "use strict";

    function toSelectorList(value) {
        if (!value) {
            return [];
        }
        return value
            .split(",")
            .map(function (item) {
                return item.trim();
            })
            .filter(Boolean);
    }

    function showTargets(selector, display) {
        toSelectorList(selector).forEach(function (item) {
            document.querySelectorAll(item).forEach(function (el) {
                if (display) {
                    el.style.display = display;
                } else {
                    el.style.display = "";
                }
            });
        });
    }

    function hideTargets(selector) {
        toSelectorList(selector).forEach(function (item) {
            document.querySelectorAll(item).forEach(function (el) {
                el.style.display = "none";
            });
        });
    }

    function resetTargets(selector) {
        toSelectorList(selector).forEach(function (item) {
            document.querySelectorAll(item).forEach(function (el) {
                if (typeof el.reset === "function") {
                    el.reset();
                } else if ("value" in el) {
                    el.value = "";
                }
            });
        });
    }

    function clearTargets(selector) {
        toSelectorList(selector).forEach(function (item) {
            document.querySelectorAll(item).forEach(function (el) {
                if ("value" in el) {
                    el.value = "";
                } else {
                    el.innerHTML = "";
                }
            });
        });
    }

    function applyAuxActions(trigger) {
        if (trigger.dataset.reset) {
            resetTargets(trigger.dataset.reset);
        }
        if (trigger.dataset.clear) {
            clearTargets(trigger.dataset.clear);
        }
    }

    function handleShowHide(trigger) {
        if (trigger.dataset.show) {
            showTargets(trigger.dataset.show, trigger.dataset.showDisplay);
        }
        if (trigger.dataset.hide) {
            hideTargets(trigger.dataset.hide);
        }
    }

    function handleModalOpen(trigger) {
        var target = trigger.dataset.modalOpen;
        if (!target) {
            return false;
        }
        showTargets(target, trigger.dataset.modalDisplay || "flex");
        applyAuxActions(trigger);
        return true;
    }

    function handleModalClose(trigger) {
        if (!trigger.dataset.modalClose) {
            return false;
        }
        var selector = trigger.dataset.modalTarget;
        if (selector) {
            hideTargets(selector);
        } else {
            var modal = trigger.closest(".modal");
            if (modal) {
                modal.style.display = "none";
            }
        }
        applyAuxActions(trigger);
        return true;
    }

    function handleCopy(trigger) {
        var text = trigger.dataset.copyText;
        if (!text) {
            return false;
        }

        var original = trigger.textContent;
        var successText = trigger.dataset.copySuccessText || "Copied!";
        var resetText = trigger.dataset.copyResetText || original;
        var timeout = parseInt(trigger.dataset.copyTimeout || "2000", 10);

        function updateLabel() {
            trigger.textContent = successText;
            window.setTimeout(function () {
                trigger.textContent = resetText;
            }, timeout);
        }

        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(updateLabel).catch(function () {
                updateLabel();
            });
            return true;
        }

        var textarea = document.createElement("textarea");
        textarea.value = text;
        textarea.setAttribute("readonly", "readonly");
        textarea.style.position = "absolute";
        textarea.style.left = "-9999px";
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand("copy");
            updateLabel();
        } finally {
            document.body.removeChild(textarea);
        }
        return true;
    }

    function handleAudio(trigger) {
        var selector = trigger.dataset.audioTarget;
        if (!selector) {
            return false;
        }
        var audio = document.querySelector(selector);
        if (audio && typeof audio.play === "function") {
            audio.play();
        }
        return true;
    }

    function handleParentEdit(trigger) {
        if (!trigger.dataset.parentEdit) {
            return false;
        }
        var id = trigger.dataset.parentId;
        var name = trigger.dataset.parentName || "";
        var email = trigger.dataset.parentEmail || "";
        var isAdmin = trigger.dataset.parentIsAdmin === "true";

        var form = document.getElementById("editForm");
        if (form && id) {
            form.action = "/admin/parents/" + id + "/update";
        }
        var nameInput = document.getElementById("edit_name");
        var emailInput = document.getElementById("edit_email");
        var adminInput = document.getElementById("edit_is_admin");
        if (nameInput) {
            nameInput.value = name;
        }
        if (emailInput) {
            emailInput.value = email;
        }
        if (adminInput) {
            adminInput.checked = isAdmin;
        }

        showTargets("#editModal", "flex");
        return true;
    }

    function handleKidEdit(trigger) {
        if (!trigger.dataset.kidEdit) {
            return false;
        }
        var id = trigger.dataset.kidId;
        var name = trigger.dataset.kidName || "";
        var avatar = trigger.dataset.kidAvatar || "#4A90E2";
        var username = trigger.dataset.kidUsername || "";

        var form = document.getElementById("editForm");
        if (form && id) {
            form.action = "/admin/children/" + id + "/update";
        }
        var nameInput = document.getElementById("edit_name");
        var usernameInput = document.getElementById("edit_username");
        var avatarInput = document.getElementById("edit_avatar_color");
        var passwordInput = document.getElementById("edit_password");
        if (nameInput) {
            nameInput.value = name;
        }
        if (usernameInput) {
            usernameInput.value = username;
        }
        if (avatarInput) {
            avatarInput.value = avatar;
        }
        if (passwordInput) {
            passwordInput.value = "";
        }

        showTargets("#editModal", "flex");
        return true;
    }

    function stringToColor(str) {
        var colors = ["#4A90E2", "#50C878", "#FF6B6B", "#FFA500", "#9B59B6", "#E91E63", "#00BCD4", "#FF9800"];
        var hash = 0;
        for (var i = 0; i < str.length; i++) {
            hash = str.charCodeAt(i) + ((hash << 5) - hash);
        }
        return colors[Math.abs(hash) % colors.length];
    }

    function renderRememberedUsernames() {
        var container = document.getElementById("remembered-usernames");
        if (!container) {
            return;
        }
        var remembered;
        try {
            remembered = JSON.parse(localStorage.getItem("spellingclash_usernames") || "[]");
        } catch (error) {
            remembered = [];
        }
        if (!Array.isArray(remembered) || remembered.length === 0) {
            return;
        }

        container.innerHTML = "";
        var label = document.createElement("p");
        label.className = "remembered-label";
        label.textContent = "Previously used:";
        container.appendChild(label);

        remembered.forEach(function (username) {
            var item = document.createElement("div");
            item.className = "kid-select-item remembered-username";

            var button = document.createElement("div");
            button.className = "kid-select-button";
            button.dataset.rememberedUsername = username;

            var avatar = document.createElement("div");
            avatar.className = "kid-avatar-large";
            avatar.style.backgroundColor = stringToColor(username);
            avatar.textContent = username.charAt(0).toUpperCase();

            var name = document.createElement("span");
            name.className = "kid-name";
            name.textContent = username;

            button.appendChild(avatar);
            button.appendChild(name);
            item.appendChild(button);
            container.appendChild(item);
        });
    }

    function handleRememberedUsername(trigger) {
        var username = trigger.dataset.rememberedUsername;
        if (!username) {
            return false;
        }
        var input = document.getElementById("username");
        var form = document.getElementById("username-form");
        if (input && form) {
            input.value = username;
            form.submit();
        }
        return true;
    }

    function attachMissingLetterBehavior(scope) {
        var root = scope || document;
        var form = root.querySelector("[data-missing-letter-form='true']");
        if (!form) {
            return;
        }

        var inputs = form.querySelectorAll(".letter-input");
        var hiddenInput = form.querySelector("#combined-guess");
        if (inputs.length === 0) {
            return;
        }

        inputs[0].focus();

        inputs.forEach(function (input, idx) {
            input.addEventListener("input", function () {
                input.value = input.value.toLowerCase();
                if (input.value.length === 1 && idx < inputs.length - 1) {
                    inputs[idx + 1].focus();
                }
            });

            input.addEventListener("keydown", function (event) {
                if (event.key === "Backspace" && input.value === "" && idx > 0) {
                    inputs[idx - 1].focus();
                }
            });
        });

        form.addEventListener("submit", function () {
            var letterMap = [];
            inputs.forEach(function (input) {
                var wordIndex = parseInt(input.getAttribute("data-index"), 10);
                var letter = (input.value || "").toLowerCase();
                letterMap.push({ index: wordIndex, letter: letter });
            });

            letterMap.sort(function (a, b) {
                return a.index - b.index;
            });

            var combined = "";
            letterMap.forEach(function (item) {
                combined += item.letter;
            });

            if (hiddenInput) {
                hiddenInput.value = combined;
            }
        });
    }

    function attachPracticeForm() {
        var form = document.querySelector("[data-practice-form='true']");
        if (!form) {
            return;
        }

        var startTimeInput = document.getElementById("word-start-time");
        if (startTimeInput) {
            startTimeInput.value = Date.now();
        }

        form.addEventListener("submit", function (event) {
            event.preventDefault();

            var answerInput = document.getElementById("answer-input");
            if (!answerInput) {
                return;
            }
            var answer = answerInput.value.trim();
            if (!answer) {
                return;
            }

            answerInput.disabled = true;

            var formData = new URLSearchParams();
            formData.append("answer", answer);

            fetch("/child/practice/submit", {
                method: "POST",
                headers: { "Content-Type": "application/x-www-form-urlencoded" },
                body: formData
            })
                .then(function (response) {
                    if (!response.ok) {
                        throw new Error("Failed to submit answer");
                    }
                    return response.json();
                })
                .then(function (result) {
                    var feedback = document.getElementById("feedback");
                    if (!feedback) {
                        return;
                    }
                    feedback.style.display = "block";

                    if (result.isCorrect) {
                        feedback.className = "feedback-message feedback-correct";
                        feedback.innerHTML = "<h2>Correct!</h2><p>You earned " + result.points + " points!</p>";
                    } else {
                        feedback.className = "feedback-message feedback-incorrect";
                        feedback.innerHTML = "<h2>Not quite...</h2><p>The correct spelling is: <strong>" + result.correctWord + "</strong></p>";
                    }

                    window.setTimeout(function () {
                        if (result.completed) {
                            window.location.href = "/child/practice/results";
                        } else if (result.nextWord) {
                            window.location.reload();
                        }
                    }, 2000);
                })
                .catch(function () {
                    answerInput.disabled = false;
                });
        });
    }

    function attachBulkImport() {
        var form = document.querySelector("[data-bulk-import-form='true']");
        if (!form) {
            return;
        }

        var submitBtn = document.getElementById("bulk-add-submit");
        var progressDiv = document.getElementById("bulk-import-progress");
        var progressBar = document.getElementById("progress-bar-fill");
        var progressStatus = document.getElementById("progress-status");
        var progressCount = document.getElementById("progress-count");
        var progressUrl = form.dataset.progressUrl;

        form.addEventListener("submit", function (event) {
            event.preventDefault();

            if (!submitBtn || !progressDiv || !progressBar || !progressStatus || !progressCount || !progressUrl) {
                return;
            }

            submitBtn.disabled = true;
            submitBtn.textContent = "Processing...";
            progressDiv.style.display = "block";

            var formData = new FormData(form);

            fetch(form.action, {
                method: "POST",
                body: formData
            })
                .then(function (response) {
                    if (!response.ok) {
                        throw new Error("Failed to start import");
                    }
                    return response.json();
                })
                .then(function () {
                    var pollInterval = window.setInterval(function () {
                        fetch(progressUrl)
                            .then(function (response) {
                                if (!response.ok) {
                                    throw new Error("Failed to get progress");
                                }
                                return response.json();
                            })
                            .then(function (progress) {
                                var percentage = progress.total > 0 ? (progress.processed / progress.total) * 100 : 0;
                                progressBar.style.width = percentage + "%";
                                progressCount.textContent = progress.processed + " / " + progress.total;

                                if (progress.failed > 0) {
                                    progressStatus.textContent = "Processing (" + progress.failed + " failed)...";
                                } else {
                                    progressStatus.textContent = "Processing...";
                                }

                                if (progress.completed) {
                                    window.clearInterval(pollInterval);

                                    if (progress.error) {
                                        progressStatus.textContent = "Error: " + progress.error;
                                        progressStatus.style.color = "red";
                                        submitBtn.disabled = false;
                                        submitBtn.textContent = "Add All Words";
                                    } else {
                                        progressStatus.textContent = "Complete!";
                                        progressStatus.style.color = "green";
                                        window.setTimeout(function () {
                                            window.location.reload();
                                        }, 1000);
                                    }
                                }
                            })
                            .catch(function () {
                                window.clearInterval(pollInterval);
                                progressStatus.textContent = "Error checking progress";
                                progressStatus.style.color = "red";
                                submitBtn.disabled = false;
                                submitBtn.textContent = "Add All Words";
                            });
                    }, 500);
                })
                .catch(function (error) {
                    if (progressStatus) {
                        progressStatus.textContent = "Error: " + error.message;
                        progressStatus.style.color = "red";
                    }
                    submitBtn.disabled = false;
                    submitBtn.textContent = "Add All Words";
                });
        });
    }

    function attachRememberUsernameForm() {
        var form = document.querySelector("[data-remember-username]");
        if (!form) {
            return;
        }

        form.addEventListener("submit", function () {
            var username = form.dataset.rememberUsername;
            if (!username) {
                return;
            }

            var remembered;
            try {
                remembered = JSON.parse(localStorage.getItem("spellingclash_usernames") || "[]");
            } catch (error) {
                remembered = [];
            }

            if (!Array.isArray(remembered)) {
                remembered = [];
            }

            if (remembered.indexOf(username) === -1) {
                remembered.push(username);
                localStorage.setItem("spellingclash_usernames", JSON.stringify(remembered));
            }
        });
    }

    function attachPasswordConfirm() {
        var form = document.querySelector("[data-password-confirm='true']");
        if (!form) {
            return;
        }

        form.addEventListener("submit", function (event) {
            var password = document.getElementById("password");
            var confirm = document.getElementById("confirm_password");
            if (!password || !confirm) {
                return;
            }
            if (password.value !== confirm.value) {
                event.preventDefault();
                window.alert("Passwords do not match. Please try again.");
            }
        });
    }

    document.addEventListener("click", function (event) {
        var target = event.target.closest("[data-modal-open], [data-modal-close], [data-show], [data-hide], [data-copy-text], [data-audio-target], [data-parent-edit], [data-kid-edit], [data-remembered-username]");
        if (!target) {
            if (event.target.classList && event.target.classList.contains("modal") && event.target.dataset.modalClickClose === "true") {
                event.target.style.display = "none";
            }
            return;
        }

        if (handleParentEdit(target)) {
            return;
        }
        if (handleKidEdit(target)) {
            return;
        }
        if (handleModalOpen(target)) {
            return;
        }
        if (handleModalClose(target)) {
            return;
        }
        if (handleCopy(target)) {
            return;
        }
        if (handleAudio(target)) {
            return;
        }
        if (handleRememberedUsername(target)) {
            return;
        }

        handleShowHide(target);
        applyAuxActions(target);
    });

    document.addEventListener("submit", function (event) {
        var form = event.target;
        if (!form || !(form instanceof HTMLFormElement)) {
            return;
        }
        var message = form.dataset.confirm;
        var checkboxSelector = form.dataset.confirmCheckbox;
        if (checkboxSelector) {
            var checkbox = document.querySelector(checkboxSelector);
            if (checkbox && checkbox.checked) {
                message = form.dataset.confirmCheckedMessage || message;
            } else {
                message = form.dataset.confirmMessage || message;
            }
        }
        if (message) {
            if (!window.confirm(message)) {
                event.preventDefault();
            }
        }
    });

    document.addEventListener("DOMContentLoaded", function () {
        renderRememberedUsernames();
        attachMissingLetterBehavior(document);
        attachPracticeForm();
        attachBulkImport();
        attachRememberUsernameForm();
        attachPasswordConfirm();
    });

    document.body.addEventListener("htmx:afterSwap", function (event) {
        attachMissingLetterBehavior(event.target);
    });
})();
